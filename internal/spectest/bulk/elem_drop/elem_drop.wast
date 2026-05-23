;; elem.drop
(module
  (table 1 funcref)
  (func $f)
  (elem $p funcref (ref.func $f))
  (elem $a (table 0) (i32.const 0) func $f)

  (func (export "drop_passive") (elem.drop $p))
  (func (export "init_passive") (param $len i32)
    (table.init $p (i32.const 0) (i32.const 0) (local.get $len))
  )

  (func (export "drop_active") (elem.drop $a))
  (func (export "init_active") (param $len i32)
    (table.init $a (i32.const 0) (i32.const 0) (local.get $len))
  )
)

(invoke "init_passive" (i32.const 1))
(invoke "drop_passive")
(invoke "drop_passive")
(assert_return (invoke "init_passive" (i32.const 0)))
(assert_trap (invoke "init_passive" (i32.const 1)) "out of bounds table access")
(invoke "init_passive" (i32.const 0))
(invoke "drop_active")
(assert_return (invoke "init_active" (i32.const 0)))
(assert_trap (invoke "init_active" (i32.const 1)) "out of bounds table access")
(invoke "init_active" (i32.const 0))
