(module
 (type $0 (func (param f64) (result f64)))
 (export "sin" (func $0))
 (func $0 (param $0 f64) (result f64)
  (local $1 f64)
  (local $2 f64)
  (local $3 i32)
  (local $4 i64)
  (if
   (f64.eq
    (local.get $0)
    (f64.const 0)
   )
   (then
    (return
     (local.get $0)
    )
   )
  )
  (local.set $1
   (f64.const nan:0x8000000000000)
  )
  (block $block
   (br_if $block
    (i64.gt_u
     (i64.and
      (i64.reinterpret_f64
       (local.get $0)
      )
      (i64.const 9223372036854775807)
     )
     (i64.const 9218868437227405311)
    )
   )
   (local.set $1
    (f64.add
     (local.get $0)
     (f64.mul
      (f64.convert_i64_s
       (local.tee $4
        (i64.trunc_sat_f64_s
         (f64.add
          (f64.mul
           (local.get $0)
           (f64.const 0.6366197723675814)
          )
          (f64.copysign
           (f64.const 0.5)
           (local.get $0)
          )
         )
        )
       )
      )
      (f64.const -1.5707963267948966)
     )
    )
   )
   (local.set $1
    (loop $label (result f64)
     (if (result f64)
      (f64.gt
       (f64.abs
        (local.get $1)
       )
       (f64.const 7.450580596923828e-09)
      )
      (then
       (local.set $3
        (i32.add
         (local.get $3)
         (i32.const 1)
        )
       )
       (local.set $1
        (f64.mul
         (local.get $1)
         (f64.const 0.5)
        )
       )
       (br $label)
      )
      (else
       (local.set $0
        (f64.const 1)
       )
       (block $block3
        (block $block2
         (loop $label1
          (if
           (local.get $3)
           (then
            (local.set $3
             (i32.sub
              (local.get $3)
              (i32.const 1)
             )
            )
            (local.set $2
             (f64.mul
              (local.get $1)
              (local.get $1)
             )
            )
            (local.set $1
             (f64.add
              (local.tee $0
               (f64.mul
                (local.get $0)
                (local.get $1)
               )
              )
              (local.get $0)
             )
            )
            (local.set $0
             (f64.sub
              (f64.const 1)
              (f64.add
               (local.get $2)
               (local.get $2)
              )
             )
            )
            (br $label1)
           )
           (else
            (block $block1
             (br_table $block1 $block2 $block3 $block
              (i32.sub
               (i32.and
                (i32.wrap_i64
                 (local.get $4)
                )
                (i32.const 3)
               )
               (i32.const 1)
              )
             )
            )
           )
          )
         )
         (return
          (local.get $0)
         )
        )
        (return
         (f64.neg
          (local.get $1)
         )
        )
       )
       (f64.neg
        (local.get $0)
       )
      )
     )
    )
   )
  )
  (local.get $1)
 )
)

