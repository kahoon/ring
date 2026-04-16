# ring

`ring` implements a high-performance FIFO ring buffer.

## Behavior

- `Push` appends to the back of the queue.
- `Pop` removes from the front of the queue.
- By default, a full queue drops the oldest element on `Push`.
- `WithGrowing(true)` changes that behavior so the backing store grows instead.
- `WithMinCapacity(n)` rounds `n` up to the next power of two and clamps it to at least the default minimum.
- The queue is not safe for concurrent use.
- `All()` allows in-place mutation of elements, but `Push` and `Pop` during iteration will panic.

## Fixed-Size Queue

```go
import "github.com/kahoon/ring"

q := ring.New[int](ring.WithMinCapacity[int](4))

for i := 1; i <= 4; i++ {
	drop, dropped := q.Push(i)
	fmt.Println("push", i, "dropped?", dropped, "drop", drop)
}

drop, dropped := q.Push(5)
fmt.Println("push 5 dropped?", dropped, "drop", drop)

for !q.IsEmpty() {
	v, _ := q.Pop()
	fmt.Println(v)
}
```

Output:

```text
push 1 dropped? false drop 0
push 2 dropped? false drop 0
push 3 dropped? false drop 0
push 4 dropped? false drop 0
push 5 dropped? true drop 1
2
3
4
5
```

## Growing Queue And In-Place Updates

```go
type Sprite struct {
	X int
	Y int
}

q := ring.New[*Sprite](
	ring.WithMinCapacity[*Sprite](16),
	ring.WithGrowing[*Sprite](true),
)

q.Push(&Sprite{X: 10, Y: 20})
q.Push(&Sprite{X: 30, Y: 40})

for sprite := range q.All() {
	sprite.X += 5
	sprite.Y += 5
}

front, _ := q.Peek()
fmt.Println(front.X, front.Y) // 15 25
```

If you need to enqueue or dequeue while traversing, finish the iteration first.
`All()` is for visiting the current contents of the queue, not for structurally
modifying it mid-pass.
