package ring

import (
	"testing"
)

func TestNew(t *testing.T) {
	q := New[int]()
	if q.capacity != initialCapacity {
		t.Errorf("Expected default capacity %d, got %d", initialCapacity, q.capacity)
	}

	q2 := New[int](WithMinCapacity[int](5), WithGrowing[int](true))
	if q2.capacity != initialCapacity {
		t.Errorf("Expected fallback to %d, got %d", initialCapacity, q2.capacity)
	}
	if !q2.growing {
		t.Error("Expected growing to be enabled")
	}
}

func TestWithMinCapacity(t *testing.T) {
	tests := []struct {
		name string
		min  uint64
		want uint64
	}{
		{name: "zero clamps to default", min: 0, want: initialCapacity},
		{name: "below default clamps to default", min: 7, want: initialCapacity},
		{name: "power of two is preserved", min: 8, want: 8},
		{name: "non power of two rounds up", min: 9, want: 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := New[int](WithMinCapacity[int](tt.min))
			if q.capacity != tt.want {
				t.Fatalf("Expected capacity %d, got %d", tt.want, q.capacity)
			}
		})
	}
}

func TestPushPop(t *testing.T) {
	q := New[int]()

	// Fill it up
	for i := 1; i <= initialCapacity; i++ {
		q.Push(i)
	}

	if q.Len() != initialCapacity {
		t.Errorf("Expected length %d, got %d", initialCapacity, q.Len())
	}

	// Dequeue and check order
	for i := 1; i <= initialCapacity; i++ {
		val, ok := q.Pop()
		if !ok || val != i {
			t.Errorf("Step %d: Expected %d, got %d (ok: %v)", i, i, val, ok)
		}
	}

	// Underflow check
	_, ok := q.Pop()
	if ok {
		t.Errorf("Expected false, got %v", ok)
	}
}

func TestRingBufferWrapping(t *testing.T) {
	// Create a small queue, move head/tail to the middle
	q := New[int]()
	q.Push(1)
	q.Push(2)
	q.Pop()
	q.Pop()
	q.Push(3)
	q.Push(4)
	q.Push(5)
	q.Push(6)
	q.Push(7)
	q.Push(8)

	if q.Len() != 6 {
		t.Errorf("Expected length 6, got %d", q.Len())
	}

	val, _ := q.Pop()
	if val != 3 {
		t.Errorf("Expected 3, got %d", val)
	}
}

func TestResize(t *testing.T) {
	q := New(WithGrowing[int](true))
	for i := 1; i <= initialCapacity; i++ {
		q.Push(i)
	}

	// This should trigger resize
	q.Push(initialCapacity + 1)

	if q.capacity != initialCapacity*2 {
		t.Errorf("Expected capacity to double to %d, got %d", initialCapacity*2, q.capacity)
	}

	if q.Len() != initialCapacity+1 {
		t.Errorf("Expected length %d, got %d", initialCapacity+1, q.Len())
	}

	// Ensure data integrity after resize
	for i := 1; i <= initialCapacity+1; i++ {
		val, _ := q.Pop()
		if val != i {
			t.Errorf("Post-resize: expected %d, got %d", i, val)
		}
	}
}

func TestPeek(t *testing.T) {
	q := New[string]()

	_, ok := q.Peek()
	if ok {
		t.Error("Peek on empty queue should return false")
	}

	q.Push("hello")
	val, ok := q.Peek()
	if !ok || val != "hello" {
		t.Errorf("Peek failed: expected %s, got %s", "hello", val)
	}

	// Ensure Peek didn't remove the item
	if q.Len() != 1 {
		t.Error("Peek should not change queue length")
	}
}

func TestIteration(t *testing.T) {
	q := New[int]()
	items := []int{10, 20, 30}
	for _, v := range items {
		q.Push(v)
	}

	sum := 0
	for v := range q.All() {
		sum += *v
		*v = *v + 1 // Test mutation via pointer
	}

	if sum != 60 {
		t.Errorf("All sum expected 60, got %d", sum)
	}

	val, _ := q.Pop()
	if val != 11 {
		t.Errorf("Expected mutated value 11, got %d", val)
	}
}

func TestPushPanicsDuringIteration(t *testing.T) {
	q := New[int]()
	for _, v := range []int{1, 2, 3} {
		q.Push(v)
	}

	defer expectPanic(t, "ring: Push/Pop during iteration")
	for v := range q.All() {
		q.Push(*v + 10)
	}
}

func TestPopPanicsDuringIteration(t *testing.T) {
	q := New[int]()
	for _, v := range []int{1, 2, 3} {
		q.Push(v)
	}

	defer expectPanic(t, "ring: Push/Pop during iteration")
	for range q.All() {
		q.Pop()
	}
}

func TestPushDropsOldestWhenFull(t *testing.T) {
	q := New[int]()
	for i := 1; i <= initialCapacity; i++ {
		drop, dropped := q.Push(i)
		if dropped {
			t.Fatalf("Unexpected drop while filling queue: %d", drop)
		}
	}

	drop, dropped := q.Push(initialCapacity + 1)
	if !dropped {
		t.Fatal("Expected push to drop the oldest item when full")
	}
	if drop != 1 {
		t.Fatalf("Expected dropped item 1, got %d", drop)
	}

	for want := 2; want <= initialCapacity+1; want++ {
		got, ok := q.Pop()
		if !ok {
			t.Fatalf("Expected value %d, queue was empty", want)
		}
		if got != want {
			t.Fatalf("Expected value %d, got %d", want, got)
		}
	}
}

func TestMemoryLeakPrevention(t *testing.T) {
	// Using pointers to check if Pop clears the reference
	type item struct{ name string }
	q := New[*item]()
	obj := &item{"test"}
	q.Push(obj)
	q.Pop()
	// Internal: Check if the head index in the underlying slice is nil
	// head was 0, Pop moves it to 1, but index 0 should be nilled
	if q.data[0] != nil {
		t.Error("Pop failed to zero out the data slot; potential memory leak")
	}
}

func expectPanic(t *testing.T, want string) {
	t.Helper()

	if got := recover(); got != want {
		t.Fatalf("Expected panic %q, got %v", want, got)
	}
}
