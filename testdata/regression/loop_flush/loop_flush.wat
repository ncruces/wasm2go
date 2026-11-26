;; Regression: pending lazy stack values pushed before a void loop must be
;; materialized before the loop label so back-edges do not re-evaluate them.
(module
  (func (export "expr_before_loop") (param $x i32) (result i32)
    (local.get $x)
    (loop $l
      (local.set $x (i32.add (local.get $x) (i32.const 1)))
      (br_if $l (i32.lt_s (local.get $x) (i32.const 5)))
    )
  )

  (func (export "cond_before_loop") (param $x i32) (result i32)
    (i32.eqz (local.get $x))
    (loop $l
      (local.set $x (i32.add (local.get $x) (i32.const 1)))
      (br_if $l (i32.lt_s (local.get $x) (i32.const 5)))
    )
  )
)
