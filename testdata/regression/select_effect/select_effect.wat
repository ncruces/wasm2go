(module
  (global $g (mut i32) (i32.const 0))
  (func $inc (result i32)
    global.get $g
    i32.const 1
    i32.add
    global.set $g
    i32.const 100)
  (func (export "test") (param $c i32) (result i32)
    call $inc
    i32.const 5
    local.get $c
    select)
  (func (export "counter") (result i32)
    global.get $g))
