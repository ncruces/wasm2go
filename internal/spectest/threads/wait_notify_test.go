package wasm2go

import (
	_ "embed"
	"sync"
	"testing"

	notify "github.com/ncruces/wasm2go/internal/spectest/threads/notify"
	wait "github.com/ncruces/wasm2go/internal/spectest/threads/wait"
)

func Test_globals(t *testing.T) {
	var wg sync.WaitGroup
	mem := &sharedMemory{data: make([]byte, 65536)}

	wg.Go(func() {
		wait := wait.New(mem)
		assert_return(t, wait.Xrun(), 0) // (assert_return (invoke "run") (i32.const 0))
	})

	wg.Go(func() {
		notify := notify.New(mem)
		assert_return(t, notify.Xnotify_0_beehrg(), 0) //  (assert_return (invoke "notify-0") (i32.const 0))
		notify.Xnotify_1_while_mqneum()                //  (assert_return (invoke "notify-1-while"))
	})

	wg.Wait()
}

type Memory = interface {
	Slice() *[]byte
	Waiters() *sync.Map
	Grow(delta, max int64) int64
}

type sharedMemory struct {
	wait sync.Map
	data []byte
}

func (m *sharedMemory) Xshared() Memory    { return m }
func (m *sharedMemory) Slice() *[]byte     { return &m.data }
func (m *sharedMemory) Waiters() *sync.Map { return &m.wait }
func (m *sharedMemory) Grow(delta, _ int64) int64 {
	if delta == 0 {
		return int64(len(m.data) >> 16)
	}
	return -1
}

func assert_return[T comparable](t *testing.T, got, want T) {
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
