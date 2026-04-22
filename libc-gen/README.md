# C standard library generator

`libc-gen` is a utility that provides a minimal libc
that helps translate C projects to Go (via `wasm2go`)
using `clang --target=wasm32 -nostdlib`.

It was created primarily to support translating SQLite,
but it can easily be used to adapt other C projects.

## Overview

Compiling C to Wasm without a standard library leaves a gap:
modules still need memory allocation, basic algorithms,
and host-provided capabilities.

`libc-gen` bridges this gap with two components:

1. **C sources**: a minimal C library containing header files,
and some bits best implemented in C,
such as a custom `qsort` implementation,
Doug Lea's `malloc` (or a simple bump allocator), etc.

2. **Go host functions**: a code generator that emits,
testable Go methods to back functions that the C code
expects the host to provide.

```
Usage of libc-gen:
  -c-out string
        extract libc C source and header files to directory
  -deref-mem
        dereference memory (*m.memory instead of m.memory)
  -m64
        use 64-bit pointers (int64)
  -o string
        output file (default stdout)
  -pkg string
        package name (default "main")
```
