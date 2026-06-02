package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/internal/spectest"
	linking_15 "github.com/ncruces/wasm2go/internal/spectest/linking/linking.15"
	linking_16 "github.com/ncruces/wasm2go/internal/spectest/linking/linking.16"
	linking_17 "github.com/ncruces/wasm2go/internal/spectest/linking/linking.17"
	linking_29 "github.com/ncruces/wasm2go/internal/spectest/linking/linking.29"
	linking_30 "github.com/ncruces/wasm2go/internal/spectest/linking/linking.30"
	linking_31 "github.com/ncruces/wasm2go/internal/spectest/linking/linking.31"
	linking_5 "github.com/ncruces/wasm2go/internal/spectest/linking/linking.5"
	linking_6 "github.com/ncruces/wasm2go/internal/spectest/linking/linking.6"
)

func Test_globals(t *testing.T) {
	mg := linking_5.New()
	ng := linking_6.New(mg)

	assert_return(t, *mg.Xglob(), 42)          // (assert_return (get $Mg "glob") (i32.const 42))
	assert_return(t, *ng.XMg_glob_xuy5e(), 42) // (assert_return (get $Ng "Mg.glob") (i32.const 42))
	assert_return(t, *ng.Xglob(), 43)          // (assert_return (get $Ng "glob") (i32.const 43))
	assert_return(t, mg.Xget(), 42)            // (assert_return (invoke $Mg "get") (i32.const 42))
	assert_return(t, ng.XMg_get_gon6ax(), 42)  // (assert_return (invoke $Ng "Mg.get") (i32.const 42))
	assert_return(t, ng.Xget(), 43)            // (assert_return (invoke $Ng "get") (i32.const 43))

	assert_return(t, *mg.Xmut_glob(), 142)           // (assert_return (get $Mg "mut_glob") (i32.const 142))
	assert_return(t, *ng.XMg_mut_glob_203k69(), 142) // (assert_return (get $Ng "Mg.mut_glob") (i32.const 142))
	assert_return(t, mg.Xget_mut(), 142)             // (assert_return (invoke $Mg "get_mut") (i32.const 142))
	assert_return(t, ng.XMg_get_mut_rknwmc(), 142)   // (assert_return (invoke $Ng "Mg.get_mut") (i32.const 142))

	mg.Xset_mut(241)                                 // (assert_return (invoke $Mg "set_mut" (i32.const 241)))
	assert_return(t, *mg.Xmut_glob(), 241)           // (assert_return (get $Mg "mut_glob") (i32.const 241))
	assert_return(t, *ng.XMg_mut_glob_203k69(), 241) // (assert_return (get $Ng "Mg.mut_glob") (i32.const 241))
	assert_return(t, mg.Xget_mut(), 241)             // (assert_return (invoke $Mg "get_mut") (i32.const 241))
	assert_return(t, ng.XMg_get_mut_rknwmc(), 241)   // (assert_return (invoke $Ng "Mg.get_mut") (i32.const 241))
}

func Test_tables(t *testing.T) {
	mt := linking_15.New()
	nt := linking_16.New(mt)

	assert_return(t, mt.Xcall(2), 4)                // (assert_return (invoke $Mt "call" (i32.const 2)) (i32.const 4))
	assert_return(t, nt.XMt_call_6jzka2(2), 4)      // (assert_return (invoke $Nt "Mt.call" (i32.const 2)) (i32.const 4))
	assert_return(t, nt.Xcall(2), 5)                // (assert_return (invoke $Nt "call" (i32.const 2)) (i32.const 5))
	assert_return(t, nt.Xcall_Mt_call_kdm7ei(2), 4) // (assert_return (invoke $Nt "call Mt.call" (i32.const 2)) (i32.const 4))

	assert_trap(t, func() { mt.Xcall(1) }, "uninitialized")                // (assert_trap (invoke $Mt "call" (i32.const 1)) "uninitialized")
	assert_trap(t, func() { nt.XMt_call_6jzka2(1) }, "uninitialized")      // (assert_trap (invoke $Nt "Mt.call" (i32.const 1)) "uninitialized")
	assert_return(t, nt.Xcall(1), 5)                                       // (assert_return (invoke $Nt "call" (i32.const 1)) (i32.const 5))
	assert_trap(t, func() { nt.Xcall_Mt_call_kdm7ei(1) }, "uninitialized") // (assert_trap (invoke $Nt "call Mt.call" (i32.const 1)) "uninitialized")

	assert_trap(t, func() { mt.Xcall(0) }, "uninitialized")                // (assert_trap (invoke $Mt "call" (i32.const 0)) "uninitialized")
	assert_trap(t, func() { nt.XMt_call_6jzka2(0) }, "uninitialized")      // (assert_trap (invoke $Nt "Mt.call" (i32.const 0)) "uninitialized")
	assert_return(t, nt.Xcall(0), 5)                                       // (assert_return (invoke $Nt "call" (i32.const 0)) (i32.const 5))
	assert_trap(t, func() { nt.Xcall_Mt_call_kdm7ei(0) }, "uninitialized") // (assert_trap (invoke $Nt "call Mt.call" (i32.const 0)) "uninitialized")

	assert_trap(t, func() { mt.Xcall(20) }, "undefined")                // (assert_trap (invoke $Mt "call" (i32.const 20)) "undefined")
	assert_trap(t, func() { nt.XMt_call_6jzka2(20) }, "undefined")      // (assert_trap (invoke $Nt "Mt.call" (i32.const 20)) "undefined")
	assert_trap(t, func() { nt.Xcall(7) }, "undefined")                 // (assert_trap (invoke $Nt "call" (i32.const 7)) "undefined")
	assert_trap(t, func() { nt.Xcall_Mt_call_kdm7ei(20) }, "undefined") // (assert_trap (invoke $Nt "call Mt.call" (i32.const 20)) "undefined")

	assert_return(t, nt.Xcall(3), -4)                       // (assert_return (invoke $Nt "call" (i32.const 3)) (i32.const -4))
	assert_trap(t, func() { nt.Xcall(4) }, "indirect call") // (assert_trap (invoke $Nt "call" (i32.const 4)) "indirect call")

	ot := linking_17.New(mt)

	assert_return(t, mt.Xcall(3), 4)                // (assert_return (invoke $Mt "call" (i32.const 3)) (i32.const 4))
	assert_return(t, nt.XMt_call_6jzka2(3), 4)      // (assert_return (invoke $Nt "Mt.call" (i32.const 3)) (i32.const 4))
	assert_return(t, nt.Xcall_Mt_call_kdm7ei(3), 4) // (assert_return (invoke $Nt "call Mt.call" (i32.const 3)) (i32.const 4))
	assert_return(t, ot.Xcall(3), 4)                // (assert_return (invoke $Ot "call" (i32.const 3)) (i32.const 4))

	assert_return(t, mt.Xcall(2), -4)                // (assert_return (invoke $Mt "call" (i32.const 2)) (i32.const -4))
	assert_return(t, nt.XMt_call_6jzka2(2), -4)      // (assert_return (invoke $Nt "Mt.call" (i32.const 2)) (i32.const -4))
	assert_return(t, nt.Xcall(2), 5)                 // (assert_return (invoke $Nt "call" (i32.const 2)) (i32.const 5))
	assert_return(t, nt.Xcall_Mt_call_kdm7ei(2), -4) // (assert_return (invoke $Nt "call Mt.call" (i32.const 2)) (i32.const -4))
	assert_return(t, ot.Xcall(2), -4)                // (assert_return (invoke $Ot "call" (i32.const 2)) (i32.const -4))

	assert_return(t, mt.Xcall(1), 6)                // (assert_return (invoke $Mt "call" (i32.const 1)) (i32.const 6))
	assert_return(t, nt.XMt_call_6jzka2(1), 6)      // (assert_return (invoke $Nt "Mt.call" (i32.const 1)) (i32.const 6))
	assert_return(t, nt.Xcall(1), 5)                // (assert_return (invoke $Nt "call" (i32.const 1)) (i32.const 5))
	assert_return(t, nt.Xcall_Mt_call_kdm7ei(1), 6) // (assert_return (invoke $Nt "call Mt.call" (i32.const 1)) (i32.const 6))
	assert_return(t, ot.Xcall(1), 6)                // (assert_return (invoke $Ot "call" (i32.const 1)) (i32.const 6))

	assert_trap(t, func() { mt.Xcall(0) }, "uninitialized")                // (assert_trap (invoke $Mt "call" (i32.const 0)) "uninitialized")
	assert_trap(t, func() { nt.XMt_call_6jzka2(0) }, "uninitialized")      // (assert_trap (invoke $Nt "Mt.call" (i32.const 0)) "uninitialized")
	assert_return(t, nt.Xcall(0), 5)                                       // (assert_return (invoke $Nt "call" (i32.const 0)) (i32.const 5))
	assert_trap(t, func() { nt.Xcall_Mt_call_kdm7ei(0) }, "uninitialized") // (assert_trap (invoke $Nt "call Mt.call" (i32.const 0)) "uninitialized")
	assert_trap(t, func() { ot.Xcall(0) }, "uninitialized")                // (assert_trap (invoke $Ot "call" (i32.const 0)) "uninitialized")

	assert_trap(t, func() { ot.Xcall(20) }, "undefined") // (assert_trap (invoke $Ot "call" (i32.const 20)) "undefined")
}

func Test_memory(t *testing.T) {
	mm := linking_29.New()
	nm := linking_30.New(mm)

	assert_return(t, mm.Xload(12), 2)           // (assert_return (invoke $Mm "load" (i32.const 12)) (i32.const 2))
	assert_return(t, nm.XMm_load_5sbr74(12), 2) // (assert_return (invoke $Nm "Mm.load" (i32.const 12)) (i32.const 2))
	assert_return(t, nm.Xload(12), 0xf2)        // (assert_return (invoke $Nm "load" (i32.const 12)) (i32.const 0xf2))

	om := linking_31.New(mm)

	assert_return(t, mm.Xload(12), 0xa7)           // (assert_return (invoke $Mm "load" (i32.const 12)) (i32.const 0xa7))
	assert_return(t, nm.XMm_load_5sbr74(12), 0xa7) // (assert_return (invoke $Nm "Mm.load" (i32.const 12)) (i32.const 0xa7))
	assert_return(t, nm.Xload(12), 0xf2)           // (assert_return (invoke $Nm "load" (i32.const 12)) (i32.const 0xf2))
	assert_return(t, om.Xload(12), 0xa7)           // (assert_return (invoke $Om "load" (i32.const 12)) (i32.const 0xa7))
}

func assert_return[T comparable](t *testing.T, got, want T) {
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func assert_trap(t *testing.T, fn func(), trap string) {
	defer spectest.RecoverTrap(t, trap)
	t.Helper()
	fn()
}
