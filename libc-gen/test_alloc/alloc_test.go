package testalloc

import (
	"testing"

	"github.com/ncruces/wasm2go/libc-gen/test_alloc/bump"
	"github.com/ncruces/wasm2go/libc-gen/test_alloc/sbrk"
	"github.com/ncruces/wasm2go/libc-gen/test_alloc/tlsf"
)

// Memory represents the Wasm memory export.
type Memory = interface {
	Slice() *[]byte
	Grow(delta, max int64) int64
}

// Allocator provides a common interface across all 3 translated C allocators.
type Allocator interface {
	Xfree(ptr int32)
	Xmalloc(size int32) int32
	Xrealloc(ptr, size int32) int32
	Xmemalign(align, size int32) int32
	Xmemory() Memory
}

func addCorpus(f *testing.F) {
	// Seed corpus to give the fuzzer a head start.
	const malloc_free = "\x00\x01\x01\x00"
	const malloc_realloc = "\x00\x02\x02\x00\x04"
	const memalign = "\x03\x01\x02\x00"
	f.Add(malloc_free)
	f.Add(malloc_realloc)
	f.Add(memalign)
}

func FuzzBump(f *testing.F) {
	addCorpus(f)
	f.Fuzz(func(t *testing.T, data string) {
		testWithAllocator(t, bump.New(), data)
	})
}

func FuzzTLSF(f *testing.F) {
	addCorpus(f)
	f.Fuzz(func(t *testing.T, data string) {
		testWithAllocator(t, tlsf.New(), data)
	})
}

func FuzzDougLea(f *testing.F) {
	addCorpus(f)
	f.Fuzz(func(t *testing.T, data string) {
		testWithAllocator(t, sbrk.New(), data)
	})
}

func testWithAllocator(t *testing.T, a Allocator, data string) {
	t.Logf("Testing allocator: %T", a)

	type activeAlloc struct {
		ptr  int32
		size int32
		id   uint32
	}
	var active []struct {
		ptr  int32
		size int32
		id   uint32
	}
	var nextId uint32 = 1

	// Process the fuzz data as a stream of allocator instructions.
	for len(data) >= 2 {
		op := data[0]
		mag := data[1]
		data = data[2:]

		switch op % 4 {
		case 0: // malloc
			size := decodeSize(mag)
			ptr := a.Xmalloc(size)
			t.Logf("malloc(%d) = 0x%x", size, ptr)

			if ptr != 0 {
				active = append(active, activeAlloc{ptr, size, nextId})
				fill(*a.Xmemory().Slice(), ptr, size, nextId)
				nextId++
			}

		case 1: // free
			if len(active) == 0 {
				continue
			}
			idx := int(mag) % len(active)
			alloc := active[idx]
			t.Logf("free(0x%x) // size: %d, id: %d", alloc.ptr, alloc.size, alloc.id)

			verify(*a.Xmemory().Slice(), alloc.ptr, alloc.size, alloc.id, t)
			a.Xfree(alloc.ptr)

			// Swap and pop to remove efficiently
			active[idx] = active[len(active)-1]
			active = active[:len(active)-1]

		case 2: // realloc
			if len(active) == 0 || len(data) == 0 {
				continue
			}
			idx := int(mag) % len(active)
			alloc := active[idx]

			verify(*a.Xmemory().Slice(), alloc.ptr, alloc.size, alloc.id, t)

			newMag := data[0]
			data = data[1:]
			newSize := decodeSize(newMag)
			if newSize == 0 {
				newSize = 1 // Avoid C99 0-size realloc ambiguities across allocators
			}

			ptr := a.Xrealloc(alloc.ptr, newSize)
			t.Logf("realloc(0x%x, %d) = 0x%x // old size: %d, id: %d", alloc.ptr, newSize, ptr, alloc.size, alloc.id)

			if ptr != 0 {
				// Check that the preserved prefix data remained intact.
				copySize := alloc.size
				if newSize < copySize {
					copySize = newSize
				}
				verify(*a.Xmemory().Slice(), ptr, copySize, alloc.id, t)

				// Re-fill the entire allocation with a new ID to track it uniquely.
				active[idx] = activeAlloc{ptr, newSize, nextId}
				fill(*a.Xmemory().Slice(), ptr, newSize, nextId)
				nextId++
			}

		case 3: // memalign
			if len(data) == 0 {
				continue
			}
			alignMag := data[0]
			data = data[1:]

			size := decodeSize(mag)
			align := decodeAlign(alignMag)

			ptr := a.Xmemalign(align, size)
			t.Logf("memalign(%d, %d) = 0x%x", align, size, ptr)
			mem := *a.Xmemory().Slice()

			if ptr != 0 {
				if uint32(ptr)&(uint32(align)-1) != 0 {
					t.Fatalf("memalign returned unaligned pointer: %x (align %x)", ptr, align)
				}
				active = append(active, activeAlloc{ptr, size, nextId})
				fill(mem, ptr, size, nextId)
				nextId++
			}
		}
	}

	// Final verification of all lingering allocations
	mem := *a.Xmemory().Slice()
	for _, alloc := range active {
		verify(mem, alloc.ptr, alloc.size, alloc.id, t)
	}
}

// decodeSize creates a biased distribution (0, tiny, small, medium, large, huge).
func decodeSize(b byte) int32 {
	switch b % 8 {
	case 0:
		return 0
	case 1:
		return int32(b)
	case 2:
		return int32(b) * 16
	case 3:
		return int32(b) * 256
	case 4:
		return int32(b) * 4096
	case 5:
		return int32(b) * 65536
	case 6:
		return 1
	case 7:
		return int32(b) | (int32(b) << 8)
	}
	return 0
}

// decodeAlign guarantees valid powers of two alignments (2 to 4096).
func decodeAlign(b byte) int32 {
	shift := (b % 12) + 1
	return 1 << shift
}

// fill injects an allocation-specific repeating pattern.
func fill(mem []byte, ptr, size int32, id uint32) {
	for i := range size {
		mem[ptr+i] = byte(id + uint32(i))
	}
}

// verify checks if the allocation's pattern was corrupted or overwritten.
func verify(mem []byte, ptr, size int32, id uint32, t *testing.T) {
	for i := range size {
		expected := byte(id + uint32(i))
		if mem[ptr+i] != expected {
			t.Fatalf("Memory corruption detected at ptr %x offset %x: expected %x, got %x", ptr, i, expected, mem[ptr+i])
		}
	}
}
