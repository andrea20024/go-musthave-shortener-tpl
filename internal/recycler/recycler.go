package recycler

// Resetter — interface constraint requiring a Reset() method.
type Resetter interface {
	Reset()
}

// Pool[T] is a generic object pool for reusable items with Reset().
type Pool[T Resetter] struct {
	items []T
}

// New creates and returns a new empty Pool.
func New[T Resetter]() *Pool[T] {
	return &Pool[T]{
		items: make([]T, 0),
	}
}

// Get returns an object from the pool. If the pool is empty, returns the
// zero value of T. The retrieved slot is zeroed to release references.
func (p *Pool[T]) Get() T {
	if len(p.items) == 0 {
		var zero T
		return zero
	}
	idx := len(p.items) - 1
	item := p.items[idx]
	p.items[idx] = *new(T) // zero the retrieved slot
	p.items = p.items[:idx]
	return item
}

// Put returns an object to the pool.
func (p *Pool[T]) Put(item T) {
	p.items = append(p.items, item)
}
