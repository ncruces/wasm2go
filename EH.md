# WebAssembly Exception Handling (EH) Translation Plan

This document outlines the design and implementation strategy for translating WebAssembly's Exception Handling proposal into Go code within `wasm2go`.

## Architectural Concept

WebAssembly exceptions are block-scoped, whereas Go's `defer`/`recover` mechanism is function-scoped. To accurately map Wasm `try`/`catch` blocks without bleeding panic recovery into unintended scopes, `try` blocks are translated into **Immediately Invoked Function Expressions (IIFEs)** (closures).

Throwing an exception translates to panicking with an `*Exception` type. The IIFE encapsulates the `defer` block that catches this panic.

### Handling Control Flow (Non-Local Jumps)

Because the `try` block is a closure, standard Wasm branches (`br`, `br_table`) that target labels *outside* the `try` block cannot use a simple `goto`. 

Instead, we use **Absolute Jump IDs**. 
- The IIFE returns `(jumpID int, caught *Exception)`.
- If the IIFE catches an exception, it executes `return 0, caught`.
- If a branch targets a label inside the IIFE, it uses a standard `goto`.
- If it targets a label outside the IIFE, it executes `return jumpID, nil`.
- A Wasm `return` instruction sets the named return variables of the outermost function and executes `return -1, nil`.

### Preserving Stack Traces

If a panic is recovered but doesn't match any of the Wasm `catch` tags (or if it's a native Go panic like a nil pointer dereference), it must be re-panicked **inside** the `defer`. If we wait to re-panic outside the IIFE, the Go runtime loses the original stack trace.

## Target Go Code Generation

The following is a conceptual sketch of what a generated `try...catch` block looks like:

```go
// t0, t1 are the expected results of the Wasm try block (if any)
var t0, t1 int32

// The IIFE isolates the defer/recover. 
// It returns the jumpID for non-local exits, and the caught exception.
if jump, caught := func() (jumpID int, caught *Exception) {
	defer func() {
		r := recover()
		if r == nil {
			return // No panic
		}
		
		if ex, ok := r.(*Exception); ok {
			// Compile-time generated check for tags this block cares about.
			// If there's a catch_all, this switch is omitted.
			switch ex.tag {
			case 0, 1: 
				jumpID = 0 
				caught = ex
				return // Swallow the panic, we are catching it!
			}
		}
		
		// If the tag didn't match (or it's a native Go panic),
		// rethrow it immediately inside the defer.
		// This preserves the original Wasm stack trace.
		panic(r) 
	}()

	// ... translated try body ...
	
	// Example: br L1 (where L1 is outside this IIFE, and its ID is 1)
	return 1, nil 
	
	// Example: Wasm return (assuming r0 is the outermost function's named return)
	// r0 = ...
	// return -1, nil

	// Normal completion
	// t0 = ...; t1 = ...
	return 0, nil
}(); jump != 0 { // Routing block (executed immediately after the IIFE)
	// Route the non-local exit
	switch jump {
	case -1:
		// We need to set `funcCompiler.nkdret = true` if we use this.
		return // Only in the outer scope (outermost Wasm function bare return)
	case 1:
		goto L1 // For all other targets in the current scope
	default:
		return jump, nil // Target is outside this scope, bubble up to next IIFE
	}
} else if caught != nil {
	// We already know it matched one of our tags, so we just route it.
	// (The compiler pushes `caught.values` to the translation stack here)
	switch caught.tag {
	case 0:
		// ... catch tag 0 body ...
	case 1:
		// ... catch tag 1 body ...
	default:
		// ... catch_all body ...
		// If there's no catch_all, repanic.
		// This should never happen, but helps reachability.
	}
}
```

## Implementation Chunks

To keep pull requests and commits reviewable, the implementation is broken down into five logical chunks:

Throughout this process, a core requirement is that **existing tests (which do not use exception handling) must continue to pass** after their Go code is regenerated. This is particularly important for Chunks 1 and 2, which modify the core control flow and function signatures of the generated code.

### Chunk 1: Exception Type, Tracking IIFE Depth, Absolute Jumps & Named Returns Refactor (DONE!)
- Add a `createExceptionType` to `module.go`, which generates `type Exception = struct { Tag *byte; Val []any }`.
- A type alias ensures the type can be used across packages (one module throws, another catches)
- Use the absolute index of the block in the compilation stack (`i` in `fn.branch(n)`) as the global unique identifier (Absolute Jump ID).
- Add an `iifeDepth int` field to `funcBlock`.
- Update `fn.branch(n)` logic to detect closure escapes: `escapes := fn.blocks.top().iifeDepth > targetBlock.iifeDepth`.
- Modify branch generation: if `escapes` is false, emit `goto Label`; if `escapes` is true, emit `return i, nil` to bubble up the Jump ID.
- Implement **Lazy Named Returns**:
  - Add a `namedReturns bool` flag to `funcCompiler`.
  - Use the new `returnVal` ID helper for the next steps.
  - When returning from the outermost function (`i == 0`), if `escapes` is true, set the flag, emit an assignment to the named variables
    (`r0, r1 = ...`), and emit `return -1, nil`. If `escapes` is false, emit a normal return (`return x, y`).
  - At the end of `readCodeForFunction`, check `fn.namedReturns` and only mutate the function signature to use named returns if required.
  - The function signature should become `func (m *Module) Xfoo(...) (r0 int32, r1 int64)`.

- **Testability:** Pure refactoring. Existing modules should generate functionally equivalent, unmodified code (no named returns on standard functions), and all current tests must pass.

### Chunk 2: Implementing `throw` (DONE!)
- Parse the Tag section (Section 13) to resolve exception types (use `externTag`).
- A Tag is pointer to a unique `byte` in memory created with `new(byte)`.
- Tags are stored in the module, and indexed; they can be imported and exported.
- Implement parsing/AST generation for the `throw` instruction, by calling `panic(&Exception{Tag: tag, Val: vals})`.

- **Testability:** Wasm modules using `throw` (without `try`) can be tested; they should successfully crash the Go program with a panic containing the exception tag and values.

### Chunk 3: Implementing `try` Blocks
- Add parsing and AST generation for `try` blocks (opcodes).
- Generate the IIFE, the `defer`/`recover` closure, and the named returns (`jumpID`, `caught`).
- Generate the routing `switch` immediately following the IIFE to handle non-local jumps.

- **Testability:** Partial. The generated code for `try` blocks will be incomplete until `catch` is implemented, so full end-to-end Wasm tests are deferred to Chunk 5.

### Chunk 4: Implementing `catch` and `catch_all`
- Generate the tag-checking `switch` inside the `defer`.
- Handle pushing the exception's payload values onto the compile-time value stack (`fn.stack`).
- Translate `catch` blocks into the post-IIFE switch cases.
- Translate `catch_all` as the fallback mechanism.

- **Testability:** Full end-to-end exception handling is testable, including nested `try` blocks, catching specific tags, `catch_all`, and complex non-local exits out of closures.