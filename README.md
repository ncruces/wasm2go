# wasm2go

This project is trying to build a Wasm to Go translator.

The input is a Wasm module, and the output is a single Go source file,
with no dependencies except for the standard library.

That file forms a self contained package,
that exports a single structure called `Module`.

The methods of this structure will be the Wasm module's exported functions.

Only a minimal set of the Wasm specification will be supported,
as the goal is to translate specific Wasm modules to Go.
For example we don't need to implement SIMD,
as we can ask our compiler to avoid emmiting it.

The current goal is to compile the `fib.wasm` module,
which exports a single function.

Because Go makes a distinction between statements and expresions,
we use a stack-to-register approach to translate Wasm to Go.
