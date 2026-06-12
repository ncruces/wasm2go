;; A br_table ladder: the dispatch jumps to a block end and control
;; falls through into the next target's code. Targets 0..2 are entered
;; only by the table and can inline into the dispatch switch; the
;; default target is also reached by the br_if, so it keeps its label.
(module
  (func (export "classify") (param $x i32) (result i32)
    (local $r i32)
    (block $b3
      (block $b2
        (block $b1
          (block $b0
            (br_table $b0 $b1 $b2 $b3 (local.get $x)))
          ;; target 0
          (local.set $r (i32.add (local.get $r) (i32.const 1))))
        ;; target 1, with an early exit past target 2
        (local.set $r (i32.add (local.get $r) (i32.const 10)))
        (br_if $b3 (i32.eq (local.get $r) (i32.const 11))))
      ;; target 2
      (local.set $r (i32.add (local.get $r) (i32.const 100))))
    ;; default target
    (i32.add (local.get $r) (i32.const 1000))))
