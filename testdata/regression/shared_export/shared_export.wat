;; Regression: owned shared memory exported as "memory" must compile.
(module
  (memory (export "memory") 1 1 shared)
  (func (export "wait0") (result i32)
    (memory.atomic.wait32 (i32.const 0) (i32.const 0) (i64.const 0)))
)
