package simplelog

import "sync"

const (
	_size = 1024
)

type BufferPool struct {
	pool *sync.Pool
}

var (
	_BufferPool = NewBufferPool()
)

func NewBufferPool() BufferPool {
	return BufferPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return &Buffer{buf: make([]byte, 0, _size)}
			},
		},
	}
}

func (bp BufferPool) Get() *Buffer {
	buf := bp.pool.Get().(*Buffer)
	buf.Reset()
	return buf
}

func (bp BufferPool) Put(buf *Buffer) {
	bp.pool.Put(buf)
}

type Buffer struct {
	buf []byte
}

func (b *Buffer) AppendString(v string) {
	b.buf = append(b.buf, v...)
}

func (b *Buffer) Len() int {
	return len(b.buf)
}

func (b *Buffer) Bytes() []byte {
	return b.buf
}

func (b *Buffer) String() string {
	return string(b.buf)
}

func (b *Buffer) Reset() {
	b.buf = b.buf[:0]
}
