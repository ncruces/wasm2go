package wasm2go

// Provided declarations whose names collide with debug names in name_collision.wat.
func (m *Module) _helper() int32 { return 0 }
func _plain() int32              { return 0 }
