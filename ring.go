package ring

const initialCapacity = 1 << 3

// Queue implements a high-performance FIFO ring buffer.
// It is NOT thread-safe; use a mutex if accessing from multiple goroutines.
type Queue[T any] struct {
	data     []T
	head     uint64
	tail     uint64
	capacity uint64
	mask     uint64
	growing  bool
	iter     int
}

// Option defines a functional option for configuring the Queue.
type Option[T any] func(*Queue[T])

// WithMinCapacity configures the queue's minimum backing-store capacity.
// The requested capacity is rounded up to the next power of two and clamped
// to at least initialCapacity.
func WithMinCapacity[T any](capacity uint64) Option[T] {
	return func(q *Queue[T]) {
		q.capacity = roundUpToPowerOfTwo(capacity)
	}
}

func WithGrowing[T any](enabled bool) Option[T] {
	return func(q *Queue[T]) {
		q.growing = enabled
	}
}

// New creates a new Queue with optional configurations. By default, it starts with a capacity of
// initialCapacity and does not grow automatically.
func New[T any](opt ...Option[T]) *Queue[T] {
	q := &Queue[T]{}
	for _, o := range opt {
		o(q)
	}
	q.capacity = max(q.capacity, initialCapacity)
	q.data = make([]T, q.capacity)
	q.mask = q.capacity - 1
	return q
}

// Enqueue adds an element to the back of the queue.
// It automatically resizes if the buffer is full.
func (q *Queue[T]) Push(value T) (drop T, dropped bool) {
	q.check()
	dropped = false
	// Check if the queue is full before adding the new element
	if q.IsFull() {
		if q.growing {
			q.grow()
		} else {
			drop, dropped = q.Pop()
		}
	}
	q.data[q.tail&q.mask] = value
	q.tail++
	return drop, dropped
}

// Dequeue removes and returns the element at the front of the queue.
// Returns an false if the queue is empty.
func (q *Queue[T]) Pop() (T, bool) {
	q.check()
	var zero T // Zero value for type T
	if q.IsEmpty() {
		return zero, false
	}
	value := q.data[q.head&q.mask]
	q.data[q.head&q.mask] = zero
	q.head++
	return value, true
}

// All returns an iterator for the range keyword.
// Usage: for v := range q.All() { ... }
//
// Iteration snapshots the queue contents when iteration begins. The callback
// may mutate elements in place through *T. Structural mutation of the queue
// during iteration is unsupported; calling Push or Pop while iterating will
// panic.
func (q *Queue[T]) All() func(f func(*T) bool) {
	return func(f func(*T) bool) {
		head := q.head
		tail := q.tail
		q.iter++
		defer func() {
			q.iter--
		}()

		for i := head; i < tail; i++ {
			if !f(&q.data[i&q.mask]) {
				return
			}
		}
	}
}

// Peek returns the front element without removing it.
func (q *Queue[T]) Peek() (T, bool) {
	var zero T
	if q.IsEmpty() {
		return zero, false
	}
	return q.data[q.head&q.mask], true
}

// Len returns the current number of elements in the queue.
func (q *Queue[T]) Len() int {
	return int(q.tail - q.head)
}

// IsEmpty returns true if the queue has no elements.
func (q *Queue[T]) IsEmpty() bool {
	return q.Len() == 0
}

// IsFull returns true if the queue has reached its capacity.
func (q *Queue[T]) IsFull() bool {
	return q.Len() == int(q.capacity)
}

// resize doubles the capacity and realigns the head/tail pointers.
func (q *Queue[T]) grow() {
	newSize := len(q.data) << 1
	newData := make([]T, newSize)

	headIdx := q.head & q.mask
	n := copy(newData, q.data[headIdx:])
	copy(newData[n:], q.data[:headIdx])

	// Reset head/tail to match the new linear layout
	q.head = 0
	q.tail = uint64(len(q.data))
	q.data = newData
	q.capacity = uint64(newSize)
	q.mask = uint64(newSize - 1)
}

func (q *Queue[T]) check() {
	if q.iter != 0 {
		panic("ring: Push/Pop during iteration")
	}
}

// Rounds up n to the next power of two, or returns 1 if n is 0.
func roundUpToPowerOfTwo(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return n + 1
}
