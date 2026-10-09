# Security Policy

## Scope & Security Goals

`wasm2go` is primarily designed to facilitate `cgo`-free Go bindings
for existing C/C++ libraries by using WebAssembly as an intermediate representation.

### The "Rule of Two"

The project aims to help applications adhere to Chromium's
[Rule of Two](https://chromium.googlesource.com/chromium/src/+/main/docs/security/rule-of-2.md):
> Pick at most two:
> 1. unsafe implementation language
> 2. processing untrustworthy inputs
> 3. high privilege

Traditional `cgo` bindings run C/C++ directly in the host process
with full access to host memory and system capabilities.

By compiling C/C++ to Wasm and translating the result to pure Go,
native libraries are confined inside the WebAssembly software sandbox:

- **Code and data separation:** Executable code is immutable and exists outside the linear memory space.
- **Protected call stack:** The execution call stack (return addresses, local variables, frame pointers)
  is managed by the Go runtime and cannot be addressed or overwritten from linear memory.
  Classic control-flow hijacking techniques (buffer overflow exploits, ROP/JOP chains)
  cannot escape into the host.
- **Bounded linear memory:** All memory accesses are constrained to a contiguous slice.
  Accesses outside linear memory trap (`panic`) and cannot read or write arbitrary host memory.

### Usage of `unsafe`

When using the `-unsafe` flag, Go's `unsafe` package is used strictly for performance optimizations.
Usage complies with the rules of [`unsafe.Pointer`](https://pkg.go.dev/unsafe#Pointer)
and **does not** remove required memory bounds checks.

### Libc & Undefined Behavior

For `libc-gen` functions implemented in Go:
- Undefined behavior in C (such as unterminated strings or out-of-bounds lengths)
  may result in arbitrary guest behavior or a trap (`panic`).
- Crucially, guest undefined behavior cannot break the sandbox:
  host memory beyond `len(memory)` is never read or written to.

### Non-Goals & In-Sandbox Corruption

The sandbox boundary isolates the host from the guest (and guests from one another).

It does **not** make unsafe C/C++ code bug-free:
- Data *inside* linear memory can still be corrupted by vulnerabilities in the original C/C++ code.
- Logic bugs, infinite loops, memory leaks, and intentional panics/traps inside the guest module
  are outside the sandbox's protection scope.

## Reporting Policy ("As Is")

This repository is a personal project provided "as is" without warranty, maintained on a best-effort, spare-time basis.

- **No dedicated security team:** There are no SLAs for triage, response times, or fixes.
- **Disclosure expectations:** Please do not expect coordinated disclosure timelines or bounties.
- **How to report:** Sandbox escapes may be submitted via private Security Advisory or public Issue at your discretion.
  Use standard Issues for everything else.

If your organization requires certified security audits, guaranteed response timelines, or enterprise-grade support, this tool is not suitable.
