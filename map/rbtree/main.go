package main

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	RED = iota
	BLACK
)

type node struct {
	color  int
	key    int
	value  interface{}
	lchild *node
	rchild *node
	parent *node
}

func (n *node) isBlack() bool {
	return n == nil || n.color == BLACK
}

func (n *node) getRelatives() (parent, sibling, closetNephew, anotherNephew *node) {
	parent = n.parent
	if parent == nil {
		return
	}
	sibling = n.getSibling()
	closetNephew, anotherNephew = n.getNephews()
	return
}

// 将n从父亲节点下摘除
func (n *node) detachFromParent() {
	if n == nil || n.parent == nil {
		return
	}
	if n.parent.lchild == n {
		n.parent.lchild = nil
	} else {
		n.parent.rchild = nil
	}
}

// 返回两个侄子节点，离自己最近的第一个返回
func (n *node) getNephews() (closest *node, another *node) {
	p := n.parent
	if n == p.lchild {
		return p.rchild.lchild, p.rchild.rchild
	} else {
		return p.lchild.rchild, p.lchild.lchild
	}
}

func (n *node) childCount() (cnt int) {
	if n.lchild != nil {
		cnt++
	}
	if n.rchild != nil {
		cnt++
	}
	return
}

// 获取以@n为根的子树中最大的节点，即最右节点
func (n *node) maxNode() *node {
	for n != nil {
		if n.rchild == nil {
			return n
		}
		n = n.rchild
	}
	return nil
}

func (n *node) getSibling() *node {
	if n.parent == nil {
		return nil
	}
	if n.parent.lchild == n {
		return n.parent.rchild
	}
	return n.parent.lchild
}

// 让@target指向@n的父亲
func (n *node) shareParent(target *node) {
	parent := n.parent
	if target != nil {
		target.parent = parent
	}
	//说明n为根节点
	if parent == nil {
		return
	}
	if parent.lchild == n {
		parent.lchild = target
	} else {
		parent.rchild = target
	}
}

func (n *node) adjustRL() {
	n.rchild.adjustLL()
	n.adjustRR()
}

func (n *node) adjustLR() {
	n.lchild.adjustRR()
	n.adjustLL()
}

func (n *node) adjustRR() {
	rchild := n.rchild
	rchild.shareParent(rchild.lchild)
	n.shareParent(rchild)
	rchild.lchild = n
	n.parent = rchild
	n.color, rchild.color = rchild.color, n.color
}

func (n *node) adjustLL() {
	lchild := n.lchild
	lchild.shareParent(lchild.rchild)
	n.shareParent(lchild)
	lchild.rchild = n
	n.parent = lchild
	n.color, lchild.color = lchild.color, n.color
}

type RBTree struct {
	root *node
}

func NewRBTree() *RBTree {
	return &RBTree{}
}

func (rbt *RBTree) Get(key int) (interface{}, bool) {
	if target := get(rbt.root, key); target != nil {
		return target.value, true
	}
	return nil, false
}

func get(n *node, key int) *node {
	for n != nil {
		if n.key == key {
			return n
		}
		if key < n.key {
			n = n.lchild
		} else {
			n = n.rchild
		}
	}
	return nil
}

func insert(root *node, n *node) (justUpdate bool) {
	if root.key == n.key {
		root.value = n.value
		return true
	}
	if root.key > n.key {
		if root.lchild == nil {
			root.lchild = n
			n.parent = root
			return
		}
		return insert(root.lchild, n)
	} else {
		if root.rchild == nil {
			root.rchild = n
			n.parent = root
			return
		}
		return insert(root.rchild, n)
	}
}

func (rbt *RBTree) makeBalance(n *node) {
	// p是父亲节点
	p := n.parent

	// 父节点为黑色时插入红色节点不会导致失衡
	if p == nil || p.color == BLACK {
		return
	}
	// 没有爷爷，即父亲为根节点
	if p.parent == nil {
		p.color = BLACK // 让根变为黑色即可
		return
	}

	u := p.getSibling() //叔叔

	//叔叔是红色时，不需要旋转，只需要变色
	if !u.isBlack() {
		p.color = BLACK
		u.color = BLACK
		p.parent.color = RED //祖父
		// 因为祖父变为红色，如果祖祖父也是红色的话，需要继续调整
		rbt.makeBalance(p.parent)
		return
	}
	// 如果祖父无父亲说明祖父为根节点
	// 旋转过程可能会导致以祖父为根的子树结构变化
	// 如果子树根发生变化，旋转后需要相应更改rbt的根节点
	flag := p.parent.parent == nil

	// 父亲为红，叔叔为黑色，需要进行LL、RR、LR或者RL调整
	if subTreeRoot := rbt.adjust(n); flag {
		rbt.root = subTreeRoot
	}
}

func (rbt *RBTree) adjust(n *node) *node {
	//判断类型
	p := n.parent
	g := p.parent
	if n == p.lchild {
		if p == g.lchild { //LL
			g.adjustLL()
		} else { //RL
			g.adjustRL()
		}
	} else {
		if p == g.rchild { //RR
			g.adjustRR()
		} else { //LR
			g.adjustLR()
		}
	}
	return g.parent
}

// 插入数据
func (rbt *RBTree) Set(key int, value interface{}) {
	// 第一次插入情况
	if rbt.root == nil {
		rbt.root = &node{
			key:   key,
			value: value,
			color: RED,
		}
		return
	}
	n := &node{
		key:   key,
		value: value,
		color: RED,
	}
	// Set数据时可能key已经存在，这时是更新操作，不会出现失衡
	if justUpdate := insert(rbt.root, n); justUpdate {
		return
	}
	// 检查并保持平衡
	rbt.makeBalance(n)
}

// 删除
func (rbt *RBTree) Del(key int) {
	// 先找到待删除节点
	target := get(rbt.root, key)
	if target == nil {
		return
	}
	rbt.del(target)
}

func (rbt *RBTree) del(target *node) {
	// 获取孩子个数
	cnt := target.childCount()
	switch cnt {
	case 0:
		// 删除节点就是根节点
		if target == rbt.root {
			rbt.root = nil
			return
		}
		// 删除节点是红色节点
		if target.color == RED {
			target.detachFromParent()
			return
		}
		// 删除黑色叶子节点
		rbt.delBlackLeaf(target)
	case 1: // 这时target一定是黑色，孩子一定是红色，用孩子替换target即可
		var child *node
		if target.lchild != nil {
			child = target.lchild
			target.lchild = nil
		} else {
			child = target.rchild
			target.rchild = nil
		}
		target.key, target.value = child.key, child.value
	case 2:
		// 以左子树最右孩子替换，这样就能转化为case 0或者case 1情况。
		replace := target.lchild.maxNode()
		target.key, target.value = replace.key, replace.value
		rbt.del(replace)
	}
}

// 删除无孩子的黑色节点
func (rbt *RBTree) delBlackLeaf(target *node) {
	// p父亲,s兄弟，c为离自己最近的侄子，d为另外一个侄子
	p, s, c, d := target.getRelatives()
	// 删除target
	target.detachFromParent()

	//CASE4和CASE5需要多次迭代，所以用for循环
	for target != rbt.root {
		if s.isBlack() && c.isBlack() && d.isBlack() { // CASE1、CASE4
			if !p.isBlack() { //CASE1
				p.color, s.color = BLACK, RED
				break
			}
			s.color = RED
			// CASE4中经过p的路径上黑节点个数变化
			// 因此需要以p做新一轮target向上迭代进行调整
			target = p
			if target == rbt.root {
				return
			}
			p, s, c, d = target.getRelatives()
		} else if !c.isBlack() && d.isBlack() { // CASE2
			if s == p.rchild {
				p.adjustRL()
			} else {
				p.adjustLR()
			}
			if p == rbt.root {
				rbt.root = c
			}
			s.color = BLACK
			break
		} else { // CASE3、CASE5
			if s == p.rchild {
				p.adjustRR()
			} else {
				p.adjustLL()
			}
			if p == rbt.root {
				rbt.root = s
			}
			if !d.isBlack() { //CASE3
				d.color = BLACK
				break
			}
			// 对于CASE5，需要调整后再进入CASE2或者CASE3
			s = c
			if s == p.rchild {
				c, d = s.lchild, s.rchild
			} else {
				c, d = s.rchild, s.lchild
			}
		}
	}
}

func (rbt *RBTree) ForEach(cb func(key int, val interface{})) {
	forEach(rbt.root, cb)
}

// 中序遍历能够得到已排序的序列
func forEach(n *node, cb func(key int, val interface{})) {
	if n == nil {
		return
	}
	forEach(n.lchild, cb)
	cb(n.key, n.value)
	forEach(n.rchild, cb)
}

func main() {
	rand.Seed(time.Now().Unix())
	for i := 0; i < 10000; i++ {
		test()
	}
	fmt.Println("test success!")
}

func test() {
	rbt := NewRBTree()
	var eleNum int

	for i := 0; i < 1000; i++ {
		v := rand.Int() % 1000 //随机方式存入若干个1000以内的数
		vv, ok := rbt.Get(v)
		if ok {
			if vv.(int) != v {
				panic(fmt.Sprintf("should got %d, but got %d\n", v, vv))
			}
			continue
		}
		//如果没有存储该数据
		eleNum++
		rbt.Set(v, v)
	}
	var keys []int
	rbt.ForEach(func(key int, val interface{}) {
		keys = append(keys, key)
	})
	if eleNum != len(keys) {
		panic(fmt.Sprintf("should have %d elements, but got %d\n", eleNum, len(keys)))
	}
	for i := 1; i < len(keys); i++ {
		if keys[i-1] > keys[i] {
			panic("keys are not sorted")
		}
	}
	// 生成一个0~len(keys)这些数随机排列的数组
	randArray := makeShuffedArray(len(keys))
	// 以随机顺序删除元素
	for i := 0; i < len(keys); i++ {
		rbt.Del(keys[randArray[i]])
	}
	hasEle := false
	rbt.ForEach(func(key int, val interface{}) {
		hasEle = true
	})
	if hasEle {
		panic("should have no elements")
	}
}

func makeShuffedArray(length int) []int {
	arr := make([]int, length)
	for i := 0; i < length; i++ {
		arr[i] = i
	}
	for i := length - 1; i > 0; i-- {
		v := rand.Int() % i
		arr[v], arr[i] = arr[i], arr[v]
	}
	return arr
}
