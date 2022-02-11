package main

const defaultSize = 1024

type node struct {
	key   int
	value int
	next  *node
}

type bucket node

type HashMap struct {
	buckets []bucket
	count   int
}

func NewHashMap() *HashMap {
	return &HashMap{
		buckets: make([]bucket, defaultSize),
	}
}

func (m *HashMap) hash(key int) int {
	return key % len(m.buckets)
}

func (m *HashMap) set(key int, value int, hash int) {
	m.count++
	m.buckets[hash].next = &node{
		key:   key,
		value: value,
		next:  m.buckets[hash].next,
	}
}

func (m *HashMap) Set(key int, value int) {
	if m.needGrow() {
		m.grow()
	}
	h := m.hash(key)
	for n := m.buckets[h].next; n != nil; n = n.next {
		if n.key == key {
			n.value = value
			return
		}
	}
	m.set(key, value, h)
}

func (m *HashMap) Get(key int) (value int, ok bool) {
	h := m.hash(key)
	for n := m.buckets[h].next; n != nil; n = n.next {
		if n.key == key {
			return n.value, true
		}
	}
	return
}

func (m *HashMap) Del(key int) {
	h := m.hash(key)
	for n := (*node)(&m.buckets[h]); n.next != nil; n = n.next {
		tmp := n.next
		if tmp.key == key {
			n.next = tmp.next
			m.count--
		}
	}
}

func (m *HashMap) needGrow() bool {
	return m.count >= len(m.buckets)
}

func (m *HashMap) grow() {
	oldbuckets := m.buckets
	m.buckets = make([]bucket, len(m.buckets)<<1)
	m.count = 0
	for i := 0; i < len(oldbuckets); i++ {
		bucket := oldbuckets[i]
		for n := bucket.next; n != nil; n = n.next {
			m.set(n.key, n.value, m.hash(n.key))
		}
	}
}

// func main() {
// 	m := NewHashMap()
// 	var times = 1
// 	for i := 0; i < times; i++ {
// 		m.Set(i, i+1)
// 	}
// 	for i := 0; i < times; i++ {
// 		v, ok := m.Get(i)
// 		assert(ok, "should have value for key: %d", i)
// 		assert(v == i+1, "want value %d, but got %d", i+1, v)
// 	}
// 	for i := times; i < times*4; i++ {
// 		_, ok := m.Get(i)
// 		assert(!ok, "should not exist key: %d", i)
// 	}
// 	fmt.Println("test success!")
// }

// func assert(cond bool, format string, args ...interface{}) {
// 	if !cond {
// 		log.Fatalf(format, args...)
// 	}
// }
