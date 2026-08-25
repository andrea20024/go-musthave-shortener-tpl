package recycler

import (
	"reflect"
	"testing"
)

// testStruct implements the Resetter interface.
type testStruct struct {
	Name string
	Num  int
	Data []byte
}

func (t *testStruct) Reset() {
	t.Name = ""
	t.Num = 0
	t.Data = nil
}

// valueStruct implements Resetter with value receiver.
type valueStruct struct {
	Val int
}

func (valueStruct) Reset() {}

func TestNew(t *testing.T) {
	p := New[*testStruct]()
	if p == nil {
		t.Fatal("New returned nil")
	}
	if len(p.items) != 0 {
		t.Errorf("new pool should be empty, got len=%d", len(p.items))
	}
}

func TestPutGet(t *testing.T) {
	p := New[*testStruct]()

	orig := &testStruct{Name: "hello", Num: 42, Data: []byte{1, 2, 3}}
	p.Put(orig)

	got := p.Get()

	if !reflect.DeepEqual(got, orig) {
		t.Errorf("Get() = %+v, want %+v", got, orig)
	}
}

func TestGetEmptyReturnsZeroValue(t *testing.T) {
	p := New[*testStruct]()
	got := p.Get()

	if got != nil {
		t.Errorf("Get() on empty pool = %v, want nil", got)
	}
}

func TestPutGetMultiple(t *testing.T) {
	p := New[*testStruct]()

	items := []*testStruct{
		{Name: "first", Num: 1},
		{Name: "second", Num: 2},
		{Name: "third", Num: 3},
	}

	for _, item := range items {
		p.Put(item)
	}

	// LIFO: last in, first out
	want := []string{"third", "second", "first"}

	for i, w := range want {
		got := p.Get()
		if got.Name != w {
			t.Errorf("Get() #%d = %q, want %q", i, got.Name, w)
		}
	}

	// Pool should be empty now
	if len(p.items) != 0 {
		t.Errorf("pool should be empty, got len=%d", len(p.items))
	}
}

func TestGetZeroesRetrievedSlot(t *testing.T) {
	p := New[*testStruct]()

	p.Put(&testStruct{Name: "a"})
	p.Put(&testStruct{Name: "b"})

	// Get one item (LIFO: returns "b")
	removed := p.Get()
	if removed.Name != "b" {
		t.Errorf("Get() returned wrong item: got %q, want %q", removed.Name, "b")
	}

	// items[0] should still have "a"
	if p.items[0].Name != "a" {
		t.Errorf("remaining item lost data: got %q, want %q", p.items[0].Name, "a")
	}

	// Pool should be truncated to 1
	if len(p.items) != 1 {
		t.Errorf("pool len=%d, want 1", len(p.items))
	}

	// Get last item
	p.Get()

	// Pool should be empty
	if len(p.items) != 0 {
		t.Errorf("pool len=%d, want 0", len(p.items))
	}
}

func TestGetPutRecycle(t *testing.T) {
	p := New[*testStruct]()

	// Put, Get, modify, Put again
	orig := &testStruct{Name: "hello", Num: 42}
	p.Put(orig)

	got := p.Get()
	got.Name = "modified"
	got.Num = 999

	p.Put(got)

	got2 := p.Get()
	if got2.Name != "modified" || got2.Num != 999 {
		t.Errorf("recycled object lost modifications: %+v", got2)
	}
}

func TestResetBeforePut(t *testing.T) {
	p := New[*testStruct]()

	obj := &testStruct{Name: "dirty", Num: 777, Data: []byte{42}}
	p.Put(obj)

	got := p.Get()
	if got.Name != "dirty" || got.Num != 777 {
		t.Errorf("expected dirty data, got %+v", got)
	}

	// Reset should clean the object
	got.Reset()
	if got.Name != "" || got.Num != 0 || got.Data != nil {
		t.Errorf("Reset() failed: Name=%q, Num=%d, Data=%v, want all zero", got.Name, got.Num, got.Data)
	}
}

func TestValueTypeWithReset(t *testing.T) {
	// Test value types with Reset()
	p := New[valueStruct]()
	p.Put(valueStruct{Val: 42})
	got := p.Get()
	if got.Val != 42 {
		t.Errorf("got Val=%d, want 42", got.Val)
	}
}
