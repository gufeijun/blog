package main

import (
	"fmt"
	"log"
)

const (
	stateEmpty = iota
	stateOccupied
	stateDeleted
	defaultSize = 128
)

type pair struct {
	state int // bucket状态
	key   int
	value int
}

// 每个桶存取一个键值对
type HashMap struct {
	// 桶
	buckets []pair
	// 元素个数
	count int
}

func NewHashMap() *HashMap {
	return &HashMap{
		buckets: make([]pair, defaultSize),
	}
}

// 哈希表扩容
func (hm *HashMap) grow() {
	oldbuckets := hm.buckets
	//桶数量倍增
	hm.buckets = make([]pair, len(oldbuckets)<<1)
	hm.count = 0
	for i := 0; i < len(oldbuckets); i++ {
		bucket := oldbuckets[i]
		if bucket.state == stateOccupied {
			hm.set(oldbuckets[i].key, oldbuckets[i].value)
		}
	}
}

// 负载因子>0.25时就扩容
func (hm *HashMap) needGrow() bool {
	return hm.count*4 >= len(hm.buckets)
}

// hash函数采用最简单的取模
func (hm *HashMap) hash(key int) int {
	return key % len(hm.buckets)
}

// 当执行Set操作时，碰到对应key的bucket、第一个空的bucket或者
// 已经被Del删除的bucket，将其返回
// 当执行Get、Del操作时，碰到对应key的bucket或者第一个空的bucket，将其返回
func (hm *HashMap) access(key int, inSet bool) (p *pair) {
	h := hm.hash(key)
	for i := 0; i < len(hm.buckets); i++ {
		bucket := hm.buckets[h]
		if !inSet && bucket.state == stateDeleted ||
			bucket.state == stateOccupied && bucket.key != key {
			h = (h + 1) % len(hm.buckets)
			continue
		}
		p = &hm.buckets[h]
		break
	}
	return
}

func (hm *HashMap) set(key int, value int) {
	p := hm.access(key, true)
	if p.state != stateOccupied {
		hm.count++
		p.state = stateOccupied
		p.key = key
	}
	p.value = value
}

func (hm *HashMap) Set(key int, value int) {
	if hm.needGrow() {
		hm.grow()
	}
	hm.set(key, value)
}

func (hm *HashMap) Del(key int) {
	p := hm.access(key, false)
	if p.state == stateEmpty {
		return
	}
	hm.count--
	p.state = stateDeleted
}

func (hm *HashMap) Get(key int) (val int, ok bool) {
	p := hm.access(key, false)
	if p.state == stateEmpty {
		return
	}
	return p.value, true
}

func main() {
	m := NewHashMap()
	var times = 1
	for i := 0; i < times; i++ {
		m.Set(i, i+1)
	}
	for i := 0; i < times; i++ {
		v, ok := m.Get(i)
		assert(ok, "should have value for key: %d", i)
		assert(v == i+1, "want value %d, but got %d", i+1, v)
	}
	for i := times; i < times*4; i++ {
		_, ok := m.Get(i)
		assert(!ok, "should not exist key: %d", i)
	}
	fmt.Println("test success!")
}

func assert(cond bool, format string, args ...interface{}) {
	if !cond {
		log.Fatalf(format, args...)
	}
}
