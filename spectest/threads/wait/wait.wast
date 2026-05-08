  (module
    (memory (import "mem" "shared") 1 1 shared)
    (func (export "run") (result i32)
      (memory.atomic.wait32 (i32.const 0) (i32.const 0) (i64.const -1))
    )
  )
