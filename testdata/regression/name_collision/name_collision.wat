;; Regression: debug names matching import fields, -provided declarations,
;; or another entity's mangled identifier must fall back to index names.
(module
  (import "env" "ext" (func $ext (result i32)))
  (table $env 1 funcref)
  (global $env (mut i32) (i32.const 10))
  (func $env (result i32)
    (i32.add (call $ext) (global.get $env)))
  (func $helper (result i32)
    (i32.add (call $env) (i32.const 1)))
  (func $plain (result i32)
    (i32.const 2))
  (func $a-b (result i32)
    (i32.add (call $helper) (call $plain)))
  (func $a_b_p4kuod (result i32)
    (i32.add (call $a-b) (i32.const 4)))
  (func (export "test") (result i32)
    (i32.add (call $a_b_p4kuod) (table.size $env)))
)
