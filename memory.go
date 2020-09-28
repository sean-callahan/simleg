package simleg

import (
	"math/rand"
	"sync"
	"time"
)

var random *rand.Rand

func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

const BlockSize = 1 << 10 // 1KB

type Memory struct {
	mu     sync.Mutex
	blocks map[uint64]*memoryBlock
}

func (m *Memory) getOrMakeBlock(addr uint64) (b *memoryBlock) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.blocks == nil {
		m.blocks = make(map[uint64]*memoryBlock)
	}
	k := addr / BlockSize
	if b, ok := m.blocks[k]; ok {
		return b
	}
	b = &memoryBlock{off: k * BlockSize}
	random.Read(b.data[:])
	m.blocks[k] = b
	return b
}

func (m *Memory) Read(b []byte, addr uint64) (n uint64, err error) {
	total := uint64(len(b))
	for n < total {
		bk := m.getOrMakeBlock(addr + n)
		off := (addr + n) % BlockSize
		end := off + total - n
		if end > BlockSize {
			end = BlockSize
		}
		n += uint64(copy(b[n:], bk.data[off:end]))
	}
	return n, nil
}

func (m *Memory) Write(b []byte, addr uint64) (n uint64, err error) {
	total := uint64(len(b))
	for n < total {
		bk := m.getOrMakeBlock(addr + n)
		off := (addr + n) % BlockSize
		end := off + total - n
		if end > BlockSize {
			end = BlockSize
		}
		n += uint64(copy(bk.data[off:end], b[n:]))
	}
	return n, nil
}

type memoryBlock struct {
	off  uint64
	data [BlockSize]byte
}
