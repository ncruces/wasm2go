(module $trig.wasm
 (type $0 (func (param f64) (result f64)))
 (global $__stack_pointer (mut i32) (i32.const 65536))
 (memory $0 1)
 (export "memory" (memory $0))
 (export "sin" (func $sin))
 (func $sin (param $0 f64) (result f64)
  (local $1 f64)
  (local $2 i64)
  (local $3 i32)
  (local $4 f64)
  (block $block
   (br_if $block
    (f64.ne
     (local.get $0)
     (f64.const 0)
    )
   )
   (return
    (local.get $0)
   )
  )
  (local.set $1
   (f64.const nan:0x8000000000000)
  )
  (block $block1
   (br_if $block1
    (i64.gt_s
     (i64.and
      (i64.reinterpret_f64
       (local.get $0)
      )
      (i64.const 9223372036854775807)
     )
     (i64.const 9218868437227405311)
    )
   )
   (block $block3
    (block $block2
     (br_if $block2
      (i32.eqz
       (f64.lt
        (f64.abs
         (local.tee $1
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
        (f64.const 9223372036854775808)
       )
      )
     )
     (local.set $2
      (i64.trunc_f64_s
       (local.get $1)
      )
     )
     (br $block3)
    )
    (local.set $2
     (i64.const -9223372036854775808)
    )
   )
   (local.set $1
    (f64.add
     (local.get $0)
     (f64.mul
      (f64.convert_i64_s
       (local.get $2)
      )
      (f64.const -1.5707963267948966)
     )
    )
   )
   (local.set $3
    (i32.const 0)
   )
   (loop $label1
    (block $block4
     (br_if $block4
      (f64.gt
       (f64.abs
        (local.get $1)
       )
       (f64.const 7.450580596923828e-09)
      )
     )
     (local.set $4
      (f64.const 1)
     )
     (block $block8
      (block $block7
       (block $block6
        (loop $label
         (block $block5
          (br_if $block5
           (local.get $3)
          )
          (br_table $block1 $block6 $block7 $block8 $block1
           (i32.and
            (i32.wrap_i64
             (local.get $2)
            )
            (i32.const 3)
           )
          )
         )
         (local.set $3
          (i32.add
           (local.get $3)
           (i32.const -1)
          )
         )
         (local.set $0
          (f64.mul
           (local.get $1)
           (local.get $1)
          )
         )
         (local.set $1
          (f64.add
           (local.tee $1
            (f64.mul
             (local.get $4)
             (local.get $1)
            )
           )
           (local.get $1)
          )
         )
         (local.set $4
          (f64.sub
           (f64.const 1)
           (f64.add
            (local.get $0)
            (local.get $0)
           )
          )
         )
         (br $label)
        )
       )
       (return
        (local.get $4)
       )
      )
      (return
       (f64.neg
        (local.get $1)
       )
      )
     )
     (local.set $1
      (f64.neg
       (local.get $4)
      )
     )
     (br $block1)
    )
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
    (br $label1)
   )
  )
  (local.get $1)
 )
)

