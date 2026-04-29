# C standard library generator

`libc-gen` is a utility that provides a minimal C standard library
to aid in translating C projects to Go (via `wasm2go`)
using `clang --target=wasm32 -nostdlib -ffreestanding`.

While primarily developed for translating SQLite,
it is suitable for porting other C projects.

## Overview

Compiling C to Wasm without a standard library leaves a gap:
modules still need memory allocation, basic algorithms,
and host-provided capabilities.

`libc-gen` bridges this gap with two components:

1. **C sources**: a minimal C library containing header files,
and some bits best implemented in C,
such as a custom `qsort` implementation, STB's `sprintf`,
Doug Lea's `malloc` (or a simple bump allocator).

2. **Go host functions**: a code generator that emits testable
Go methods to back C functions best implemented in Go.

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

## Future development

This library aims to provide a minimal implementation
of the C standard library.

Functions added to it should be part of the C standard
(excluding POSIX extensions).

Besides, the C component should not grow much beyond:
- _macros_ and _function declarations_ added to header files;
- _simple one-liners_ added to source files.

The big exception was `malloc`, as it is best implemented in C,
and Doug Lea's public domain `malloc` is excellent for Wasm;
the other was a basic `qsort` due to its use of function pointers.

The Go component will contain stuff that's best implemented in Go:
- `math.h` for `double` using package `math`;
- `string.h` because `bytes.IndexByte` is unbeatable.

I will not be adding file I/O to this, or any other OS stuff.

I _can_ add the function declarations,
and other standard C stuff you might need
(though consider adding them yourself).

But you will need to implement the Go component within your own host module.

Or just bite the bullet, use WASI and implement your own WASI host module.
