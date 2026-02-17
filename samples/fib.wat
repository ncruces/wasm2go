(module
 (type $0 (func (param i32) (result i32)))
 (export "fibonacci" (func $0))
 (func $0 (param $0 i32) (result i32)
  (local $1 i32)
  (local $2 i32)
  (local.set $0
   (select
    (local.get $0)
    (i32.const 0)
    (i32.gt_s
     (local.get $0)
     (i32.const 0)
    )
   )
  )
  (local.set $1
   (i32.const 1)
  )
  (loop $label (result i32)
   (if (result i32)
    (local.get $0)
    (then
     (local.set $0
      (i32.sub
       (local.get $0)
       (i32.const 1)
      )
     )
     (local.set $1
      (i32.add
       (local.get $2)
       (local.tee $2
        (local.get $1)
       )
      )
     )
     (br $label)
    )
    (else
     (local.get $2)
    )
   )
  )
 )
)

