package helpers

import (
	"container/list"
	"math"
	"math/bits"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Use nosplit only on functions with no loops.

//go:nosplit
func atomic_fence(mem []byte) {
	ptr := (*uint32)(unsafe.Pointer((*[4]byte)(mem)))
	atomic.AddUint32(ptr, 0)
}

//go:nosplit
func atomic_load32[T uint32 | int64](mem []byte, addr T) uint32 {
	ptr := atomic_ptr32(mem, addr)
	val := atomic.LoadUint32(ptr)
	if big {
		return bits.ReverseBytes32(val)
	}
	return val
}

//go:nosplit
func atomic_load64[T uint32 | int64](mem []byte, addr T) uint64 {
	ptr := atomic_ptr64(mem, addr)
	val := atomic.LoadUint64(ptr)
	if big {
		return bits.ReverseBytes64(val)
	}
	return val
}

//go:nosplit
func atomic_store32[T uint32 | int64](mem []byte, addr T, val uint32) {
	ptr := atomic_ptr32(mem, addr)
	if big {
		val = bits.ReverseBytes32(val)
	}
	atomic.StoreUint32(ptr, val)
}

//go:nosplit
func atomic_store64[T uint32 | int64](mem []byte, addr T, val uint64) {
	ptr := atomic_ptr64(mem, addr)
	if big {
		val = bits.ReverseBytes64(val)
	}
	atomic.StoreUint64(ptr, val)
}

//go:nosplit
func atomic_xchg32[T uint32 | int64](mem []byte, addr T, val uint32) uint32 {
	ptr := atomic_ptr32(mem, addr)
	if big {
		val = bits.ReverseBytes32(val)
	}
	val = atomic.SwapUint32(ptr, val)
	if big {
		val = bits.ReverseBytes32(val)
	}
	return val
}

//go:nosplit
func atomic_xchg64[T uint32 | int64](mem []byte, addr T, val uint64) uint64 {
	ptr := atomic_ptr64(mem, addr)
	if big {
		val = bits.ReverseBytes64(val)
	}
	val = atomic.SwapUint64(ptr, val)
	if big {
		val = bits.ReverseBytes64(val)
	}
	return val
}

func atomic_cmpxchg32[T uint32 | int64](mem []byte, addr T, old, new uint32) uint32 {
	ptr := atomic_ptr32(mem, addr)
	exp := old
	if big {
		exp = bits.ReverseBytes32(old)
		new = bits.ReverseBytes32(new)
	}
	for {
		if atomic.CompareAndSwapUint32(ptr, exp, new) {
			return old
		}
		if cur := atomic.LoadUint32(ptr); cur != exp {
			if big {
				return bits.ReverseBytes32(cur)
			}
			return cur
		}
	}
}

func atomic_cmpxchg64[T uint32 | int64](mem []byte, addr T, old, new uint64) uint64 {
	ptr := atomic_ptr64(mem, addr)
	exp := old
	if big {
		exp = bits.ReverseBytes64(old)
		new = bits.ReverseBytes64(new)
	}
	for {
		if atomic.CompareAndSwapUint64(ptr, exp, new) {
			return old
		}
		if cur := atomic.LoadUint64(ptr); cur != exp {
			if big {
				return bits.ReverseBytes64(cur)
			}
			return cur
		}
	}
}

func atomic_add32[T uint32 | int64](mem []byte, addr T, val uint32) uint32 {
	ptr := atomic_ptr32(mem, addr)
	if little {
		return atomic.AddUint32(ptr, +val) - val
	}
	for {
		cur := atomic.LoadUint32(ptr)
		old := bits.ReverseBytes32(cur)
		if atomic.CompareAndSwapUint32(ptr, cur, bits.ReverseBytes32(old+val)) {
			return old
		}
	}
}

func atomic_add64[T uint32 | int64](mem []byte, addr T, val uint64) uint64 {
	ptr := atomic_ptr64(mem, addr)
	if little {
		return atomic.AddUint64(ptr, +val) - val
	}
	for {
		cur := atomic.LoadUint64(ptr)
		old := bits.ReverseBytes64(cur)
		if atomic.CompareAndSwapUint64(ptr, cur, bits.ReverseBytes64(old+val)) {
			return old
		}
	}
}

func atomic_sub32[T uint32 | int64](mem []byte, addr T, val uint32) uint32 {
	ptr := atomic_ptr32(mem, addr)
	if little {
		return atomic.AddUint32(ptr, -val) + val
	}
	for {
		cur := atomic.LoadUint32(ptr)
		old := bits.ReverseBytes32(cur)
		if atomic.CompareAndSwapUint32(ptr, cur, bits.ReverseBytes32(old-val)) {
			return old
		}
	}
}

func atomic_sub64[T uint32 | int64](mem []byte, addr T, val uint64) uint64 {
	ptr := atomic_ptr64(mem, addr)
	if little {
		return atomic.AddUint64(ptr, -val) + val
	}
	for {
		cur := atomic.LoadUint64(ptr)
		old := bits.ReverseBytes64(cur)
		if atomic.CompareAndSwapUint64(ptr, cur, bits.ReverseBytes64(old-val)) {
			return old
		}
	}
}

//go:nosplit
func atomic_load8[T uint32 | int64](mem []byte, addr T) uint8 {
	ptr, shift := atomic_ptr8(mem, addr)
	v := atomic.LoadUint32(ptr)
	if big {
		v = bits.ReverseBytes32(v)
	}
	return uint8(v >> shift)
}

//go:nosplit
func atomic_load16[T uint32 | int64](mem []byte, addr T) uint16 {
	ptr, shift := atomic_ptr16(mem, addr)
	v := atomic.LoadUint32(ptr)
	if big {
		v = bits.ReverseBytes32(v)
	}
	return uint16(v >> shift)
}

func atomic_store8[T uint32 | int64](mem []byte, addr T, val uint8) {
	ptr, shift := atomic_ptr8(mem, addr)

	mval := uint32(val) << shift
	mask := uint32(255) << shift
	if big {
		mval = bits.ReverseBytes32(mval)
		mask = bits.ReverseBytes32(mask)
	}

	for {
		cur := atomic.LoadUint32(ptr)
		if atomic.CompareAndSwapUint32(ptr, cur, (cur&^mask)|mval) {
			return
		}
	}
}

func atomic_store16[T uint32 | int64](mem []byte, addr T, val uint16) {
	ptr, shift := atomic_ptr16(mem, addr)

	mval := uint32(val) << shift
	mask := uint32(65535) << shift
	if big {
		mval = bits.ReverseBytes32(mval)
		mask = bits.ReverseBytes32(mask)
	}

	for {
		cur := atomic.LoadUint32(ptr)
		if atomic.CompareAndSwapUint32(ptr, cur, (cur&^mask)|mval) {
			return
		}
	}
}

func atomic_xchg8[T uint32 | int64](mem []byte, addr T, val uint8) uint8 {
	ptr, shift := atomic_ptr8(mem, addr)

	mval := uint32(val) << shift
	mask := uint32(255) << shift
	if big {
		mval = bits.ReverseBytes32(mval)
		mask = bits.ReverseBytes32(mask)
	}

	for {
		cur := atomic.LoadUint32(ptr)
		if atomic.CompareAndSwapUint32(ptr, cur, (cur&^mask)|mval) {
			if big {
				cur = bits.ReverseBytes32(cur)
			}
			return uint8(cur >> shift)
		}
	}
}

func atomic_xchg16[T uint32 | int64](mem []byte, addr T, val uint16) uint16 {
	ptr, shift := atomic_ptr16(mem, addr)

	mval := uint32(val) << shift
	mask := uint32(65535) << shift
	if big {
		mval = bits.ReverseBytes32(mval)
		mask = bits.ReverseBytes32(mask)
	}

	for {
		cur := atomic.LoadUint32(ptr)
		if atomic.CompareAndSwapUint32(ptr, cur, (cur&^mask)|mval) {
			if big {
				cur = bits.ReverseBytes32(cur)
			}
			return uint16(cur >> shift)
		}
	}
}

func atomic_cmpxchg8[T uint32 | int64](mem []byte, addr T, old, new uint8) uint8 {
	ptr, shift := atomic_ptr8(mem, addr)

	mold := uint32(old) << shift
	mnew := uint32(new) << shift
	mask := uint32(255) << shift
	if big {
		mold = bits.ReverseBytes32(mold)
		mnew = bits.ReverseBytes32(mnew)
		mask = bits.ReverseBytes32(mask)
	}

	for {
		cur := atomic.LoadUint32(ptr)
		if cur&mask != mold {
			if big {
				cur = bits.ReverseBytes32(cur)
			}
			return uint8(cur >> shift)
		}
		if atomic.CompareAndSwapUint32(ptr, cur, (cur&^mask)|mnew) {
			return old
		}
	}
}

func atomic_cmpxchg16[T uint32 | int64](mem []byte, addr T, old, new uint16) uint16 {
	ptr, shift := atomic_ptr16(mem, addr)

	mold := uint32(old) << shift
	mnew := uint32(new) << shift
	mask := uint32(65535) << shift
	if big {
		mold = bits.ReverseBytes32(mold)
		mnew = bits.ReverseBytes32(mnew)
		mask = bits.ReverseBytes32(mask)
	}

	for {
		cur := atomic.LoadUint32(ptr)
		if cur&mask != mold {
			if big {
				cur = bits.ReverseBytes32(cur)
			}
			return uint16(cur >> shift)
		}
		if atomic.CompareAndSwapUint32(ptr, cur, (cur&^mask)|mnew) {
			return old
		}
	}
}

func atomic_notify[T uint32 | int64](mem []byte, addr T, count int32, waiters *sync.Map) int32 {
	_ = atomic_ptr32(mem, addr)

	wa, ok := waiters.Load(int64(addr))
	if !ok {
		return 0
	}

	w := wa.(*atomic_waiters)
	w.Lock()
	defer w.Unlock()

	if w.List == nil {
		return 0
	}

	var res uint32
	for res < uint32(count) && w.Len() > 0 {
		close(w.Remove(w.Front()).(chan struct{}))
		res++
	}
	return int32(res)
}

func atomic_wait32[T uint32 | int64](mem []byte, addr T, exp uint32, timeout int64, waiters *sync.Map) int32 {
	const (
		ok        = 0
		not_equal = 1
		timed_out = 2
	)

	ptr := atomic_ptr32(mem, addr)

	wa, _ := waiters.LoadOrStore(int64(addr), &atomic_waiters{})
	w := wa.(*atomic_waiters)

	w.Lock()
	cur := atomic.LoadUint32(ptr)
	if big {
		cur = bits.ReverseBytes32(cur)
	}

	switch {
	case cur != exp:
		w.Unlock()
		return not_equal
	case w.List == nil:
		w.List = list.New()
	case w.Len() >= math.MaxUint32:
		w.Unlock()
		panic("too many waiters")
	}

	wait := make(chan struct{})
	elem := w.PushBack(wait)
	w.Unlock()

	if timeout < 0 {
		<-wait
		return ok
	}
	select {
	case <-wait:
		return ok
	case <-time.After(time.Duration(timeout)):
	}

	w.Lock()
	select {
	case <-wait:
		w.Unlock()
		return ok
	default:
		w.Remove(elem)
		w.Unlock()
		return timed_out
	}
}

func atomic_wait64[T uint32 | int64](mem []byte, addr T, exp uint64, timeout int64, waiters *sync.Map) int32 {
	const (
		ok        = 0
		not_equal = 1
		timed_out = 2
	)

	ptr := atomic_ptr64(mem, addr)

	wa, _ := waiters.LoadOrStore(int64(addr), &atomic_waiters{})
	w := wa.(*atomic_waiters)

	w.Lock()
	cur := atomic.LoadUint64(ptr)
	if big {
		cur = bits.ReverseBytes64(cur)
	}

	switch {
	case cur != exp:
		w.Unlock()
		return not_equal
	case w.List == nil:
		w.List = list.New()
	case w.Len() >= math.MaxUint32:
		w.Unlock()
		panic("too many waiters")
	}

	wait := make(chan struct{})
	elem := w.PushBack(wait)
	w.Unlock()

	if timeout < 0 {
		<-wait
		return ok
	}
	select {
	case <-wait:
		return ok
	case <-time.After(time.Duration(timeout)):
	}

	w.Lock()
	select {
	case <-wait:
		w.Unlock()
		return ok
	default:
		w.Remove(elem)
		w.Unlock()
		return timed_out
	}
}

//go:nosplit
func atomic_ptr8[T uint32 | int64](mem []byte, addr T) (ptr *uint32, shift uint32) {
	_ = mem[addr] // bounds check
	ptr = (*uint32)(unsafe.Pointer(&mem[addr&^3]))
	shift = (uint32(addr) & 3) * 8
	return
}

//go:nosplit
func atomic_ptr16[T uint32 | int64](mem []byte, addr T) (ptr *uint32, shift uint32) {
	if uint32(addr)&1 != 0 {
		panic("unaligned atomic")
	}
	_ = mem[addr+1] // bounds check
	ptr = (*uint32)(unsafe.Pointer(&mem[addr&^3]))
	shift = (uint32(addr) & 3) * 8
	return
}

//go:nosplit
func atomic_ptr32[T uint32 | int64](mem []byte, addr T) *uint32 {
	if uint32(addr)&3 != 0 {
		panic("unaligned atomic")
	}
	_ = mem[addr+3] // bounds check
	return (*uint32)(unsafe.Pointer(&mem[addr]))
}

//go:nosplit
func atomic_ptr64[T uint32 | int64](mem []byte, addr T) *uint64 {
	if uint32(addr)&7 != 0 {
		panic("unaligned atomic")
	}
	_ = mem[addr+7] // bounds check
	return (*uint64)(unsafe.Pointer(&mem[addr]))
}

type atomic_waiters = struct {
	sync.Mutex
	*list.List
}
