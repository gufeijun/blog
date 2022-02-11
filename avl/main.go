package main

import (
	"fmt"
	"math/rand"
	"time"
)

//根节点的parent为nil
type node struct {
	key    int
	value  int
	height int
	parent *node
	lchild *node
	rchild *node
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

func (n *node) getHeight() int {
	if n == nil {
		return 0
	}
	return n.height
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

// 右旋
func (n *node) adjustLL() {
	lchild := n.lchild
	n.shareParent(lchild)
	if lchild.rchild != nil {
		lchild.rchild.parent = n
	}
	n.lchild = lchild.rchild
	n.parent = lchild
	lchild.rchild = n
	// 更新高度
	n.height = max(n.lchild.getHeight(), n.rchild.getHeight()) + 1
	lchild.height = max(lchild.lchild.getHeight(), lchild.rchild.getHeight()) + 1
}

// 左旋
func (n *node) adjustRR() {
	rchild := n.rchild
	n.shareParent(rchild)
	if rchild.lchild != nil {
		rchild.lchild.parent = n
	}
	n.rchild = rchild.lchild
	n.parent = rchild
	rchild.lchild = n
	n.height = max(n.lchild.getHeight(), n.rchild.getHeight()) + 1
	// 更新高度
	rchild.height = max(rchild.lchild.getHeight(), rchild.rchild.getHeight()) + 1
}

// 先左旋后右旋
func (n *node) adjustLR() {
	n.lchild.adjustRR()
	n.adjustLL()
}

// 先右旋后左旋
func (n *node) adjustRL() {
	n.rchild.adjustLL()
	n.adjustRR()
}

// 对不平衡子树进行调整
// 返回调整后平衡子树的根节点
func (n *node) adjust() *node {
	// 判断是什么不平衡类型
	lh, rh := n.lchild.getHeight(), n.rchild.getHeight()
	if lh < rh {
		rlh, rrh := n.rchild.lchild.getHeight(), n.rchild.rchild.getHeight()
		// RR类型
		if rlh < rrh {
			n.adjustRR()
		} else { // RL类型
			n.adjustRL()
		}
	} else {
		llh, lrh := n.lchild.lchild.getHeight(), n.lchild.rchild.getHeight()
		// LL类型
		if llh > lrh {
			n.adjustLL()
		} else { // LR类型
			n.adjustLR()
		}
	}
	// 这时n节点的双亲节点就是平衡后子树的根节点
	return n.parent
}

type AVL struct {
	root *node
}

func NewAVL() *AVL {
	return &AVL{}
}

func (avl *AVL) makeBalance(n *node) {
	// 逐次更新节点n的直系父辈节点的高度，时间复杂度O(logN)
	unbalanced := adjustHeight(n)
	// 如果非平衡节点是根节点，我们还需要更改avl的root指针
	flag := unbalanced == avl.root
	//如果存在不平衡的节点,则进行调整
	if unbalanced != nil {
		if subTreeRoot := unbalanced.adjust(); flag {
			avl.root = subTreeRoot
		}
	}
}

func (avl *AVL) Set(key, value int) {
	if avl.root == nil {
		avl.root = &node{
			key:    key,
			value:  value,
			height: 1,
		}
		return
	}
	n := &node{
		key:    key,
		value:  value,
		height: 1,
	}
	// 如果已经存在Key了并更新了value，我们不需要执行后续的操作，直接返回
	if justUpdate := insert(avl.root, n); justUpdate {
		return
	}
	avl.makeBalance(n)
}

// 插入
func insert(root *node, n *node) (justUpdate bool) {
	// 更新操作
	if root.key == n.key {
		root.value = n.value
		return true
	} else if root.key < n.key {
		if root.rchild == nil {
			root.rchild = n
			n.parent = root
			return
		}
		return insert(root.rchild, n)
	} else {
		if root.lchild == nil {
			root.lchild = n
			n.parent = root
			return
		}
		return insert(root.lchild, n)
	}
}

// 更新root根节点到startLeaf路径上节点的高度。由下至上。
// 更新过程中，将找到的第一个非平衡节点返回
func adjustHeight(startLeaf *node) (unbalanced *node) {
	n := startLeaf
	if n == nil {
		return nil
	}
	for {
		lh, rh := n.lchild.getHeight(), n.rchild.getHeight()
		delta := lh - rh
		n.height = max(lh, rh) + 1
		if unbalanced == nil && delta > 1 || delta < -1 {
			unbalanced = n
		}
		// 到达根节点
		if n.parent == nil {
			return
		}
		n = n.parent
	}
}

func (avl *AVL) Get(key int) (value int, ok bool) {
	target := get(avl.root, key)
	if target == nil {
		return
	}
	return target.value, true
}

func get(n *node, key int) (target *node) {
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

/* 删除节点N
1、如果N仅有一个孩子，将孩子替代N的位置
2、如果N有两个孩子，将左子树最大值或者右子树最小值替换N，这样也可以满足左小右大的条件
3、如果N没孩子，则直接删除N即可
删除后，需要判断是否满足平衡
*/
func (avl *AVL) Del(key int) {
	target := get(avl.root, key)
	if target == nil {
		return
	}
	var needAdjustHeight *node
	if target.rchild != nil && target.lchild != nil { //有两个孩子
		// 左子树的最大节点即最右节点, 可能含有左孩子
		lTreeMaxNode := target.lchild.maxNode()
		needAdjustHeight = lTreeMaxNode.parent
		lTreeMaxNode.shareParent(lTreeMaxNode.lchild)
		// 交换节点除了可以移动指针外，也可以直接拷贝KV对
		target.key = lTreeMaxNode.key
		target.value = lTreeMaxNode.value
	} else {
		// 删除根节点，需要修改avl.root指针，单独讨论
		if target == avl.root {
			if target.lchild == nil {
				avl.root = avl.root.rchild
			} else {
				avl.root = avl.root.lchild
			}
			return
		}
		needAdjustHeight = target.parent
		if target.lchild == nil && target.rchild == nil { //没孩子
			target.detachFromParent()
		} else if target.lchild != nil { //有左孩子
			target.shareParent(target.lchild)
		} else { //有右孩子
			target.shareParent(target.rchild)
		}
	}
	// 对路径上所有可能更改高度的节点进行高度更新
	avl.makeBalance(needAdjustHeight)
}

func (avl *AVL) ForEach(cb func(key, val int)) {
	forEach(avl.root, cb)
}

// 中序遍历能够得到已排序的序列
func forEach(n *node, cb func(key, val int)) {
	if n == nil {
		return
	}
	forEach(n.lchild, cb)
	cb(n.key, n.value)
	forEach(n.rchild, cb)
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func main() {
	avl := NewAVL()
	rand.Seed(time.Now().Unix())
	// 测试1000次
	for t := 0; t < 1000; t++ {
		var eleNum int
		for i := 0; i < 10000; i++ {
			v := rand.Int() % 10000 //随机方式存入若干个10000以内的数
			vv, ok := avl.Get(v)
			if ok {
				if vv != v {
					panic(fmt.Sprintf("should got %d, but got %d\n", v, vv))
				}
				continue
			}
			//如果没有存储该数据
			eleNum++
			avl.Set(v, v)
		}
		var keys []int
		avl.ForEach(func(key, val int) {
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
			avl.Del(keys[randArray[i]])
		}
		hasEle := false
		avl.ForEach(func(key, val int) {
			hasEle = true
		})
		if hasEle {
			panic("should have no elements")
		}
	}
	fmt.Println("test success!")
}

// 洗牌
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
