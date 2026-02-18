(module
 (type $0 (func (param i64) (result i64)))
 (export "fibonacci" (func $0))
 (func $0 (param $0 i64) (result i64)
  (local $1 i64)
  (local $2 i64)
  (local.set $0
   (select
    (local.get $0)
    (i64.const 0)
    (i64.gt_s
     (local.get $0)
     (i64.const 0)
    )
   )
  )
  (local.set $1
   (i64.const 1)
  )
  (loop $label (result i64)
   (if (result i64)
    (i64.eqz
     (local.get $0)
    )
    (then
     (local.get $2)
    )
    (else
     (local.set $0
      (i64.sub
       (local.get $0)
       (i64.const 1)
      )
     )
     (local.set $1
      (i64.add
       (local.get $2)
       (local.tee $2
        (local.get $1)
       )
      )
     )
     (br $label)
    )
   )
  )
 )
)

