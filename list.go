package main

type node struct {
	data []byte
	next *node
}

type list struct {
	head   *node
	tail   *node
	length int
}

func newList() *list {
	return &list{}
}

func (l *list) hpush(n []*node) {
	if len(n) < 1 {
		return
	}
	// empty list
	if l.head == nil {
		l.tail = n[0]
	}
	for i := 0; i < len(n); i++ {
		n[i].next = l.head
		l.head = n[i]
		l.length++
	}
}

func (l *list) tpush(n []*node) {
	if len(n) < 1 {
		return
	}
	start := 0
	// empty list
	if l.head == nil {
		l.head = n[0]
		l.tail = n[0]
		start++
		l.length++
	}
	for i := start; i < len(n); i++ {
		l.tail.next = n[i]
		l.tail = n[i]
		l.length++
	}
}
