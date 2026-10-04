;; Regression: bulk memory and table operations must trap at the logical
;; size of the memory/table after memory.grow/table.grow. Growth uses append,
;; so the backing slice can have spare capacity, and s[x:y] is bounded by cap.
(module
  (memory (export "memory") 1)
  (table 1 funcref)
  (data $d "\01\02\03\04")
  (func (export "grow") (param i32) (result i32)
    local.get 0 memory.grow)
  (func (export "ld8") (param i32) (result i32)
    local.get 0 i32.load8_u)
  (func (export "fill") (param i32 i32 i32)
    local.get 0 local.get 1 local.get 2 memory.fill)
  (func (export "copy") (param i32 i32 i32)
    local.get 0 local.get 1 local.get 2 memory.copy)
  (func (export "init") (param i32 i32 i32)
    local.get 0 local.get 1 local.get 2 memory.init $d)
  (func (export "tgrow") (param i32) (result i32)
    ref.null func local.get 0 table.grow 0)
  (func (export "tfill") (param i32 i32)
    local.get 0 ref.null func local.get 1 table.fill 0)
)
