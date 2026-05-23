  (module
    (memory (import "mem" "shared") 1 1 shared)
    (func (export "notify-0") (result i32)
      (memory.atomic.notify (i32.const 0) (i32.const 0))
    )
    (func (export "notify-1-while")
      (loop
        (i32.const 1)
        (memory.atomic.notify (i32.const 0) (i32.const 1))
        (i32.ne)
        (br_if 0)
      )
    )
  )
