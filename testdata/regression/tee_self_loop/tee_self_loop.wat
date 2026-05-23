(module
  (import "env" "fn" (func $fn (param i32)))
  (func (export "tee_self_loop") (param $v i32)
    (loop $loop
      (call $fn
        (local.tee $v (local.get $v))
      )
      (br_if $loop (local.get $v))
    )
  )
)
