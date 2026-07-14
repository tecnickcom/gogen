package sfcache

// pnode is one entry held in a [pqueue], with the key it is stored under, so that
// the queue can name the victim it hands back.
type pnode[K comparable, V any] struct {
	item *entry[V]
	key  K
}

// pqueue is a min-heap of entries ordered by their own deadline (see [entry.deadline]):
// the entry a store may take is always the one at the head.
//
// The queue holds no deadline of its own, it reads the entry's, which is immutable while
// the entry is stored. Every queued entry records its own position ([entry.idx]), so an
// entry that is replaced or removed is taken out of the queue at once, in O(log n): the
// queue never holds a superseded entry and needs no compaction pass to stay bounded.
// NOTE: this is not thread-safe, it should be used within a mutex lock.
type pqueue[K comparable, V any] struct {
	nodes []pnode[K, V]
}

// len returns the number of queued entries.
func (q *pqueue[K, V]) len() int {
	return len(q.nodes)
}

// reset empties the queue and releases its backing array.
func (q *pqueue[K, V]) reset() {
	q.nodes = nil
}

// drain hands every queued entry to the given function and empties the queue. A bulk
// removal must use it rather than take the head n times, which would cost n*O(log n) of
// sifting to establish an order that is then thrown away.
func (q *pqueue[K, V]) drain(remove func(key K, item *entry[V])) {
	for _, node := range q.nodes {
		remove(node.key, node.item)
	}

	q.reset()
}

// partition removes every entry the predicate accepts, hands it to remove, and keeps the
// rest, rebuilding the heap from scratch in O(n) rather than sifting out each removal.
func (q *pqueue[K, V]) partition(expendable func(item *entry[V]) bool, remove func(key K, item *entry[V])) {
	kept := q.nodes[:0]

	for _, node := range q.nodes {
		if expendable(node.item) {
			remove(node.key, node.item)

			continue
		}

		kept = append(kept, node)
	}

	// Clear the tail: entries left in the slice's spare capacity would stay reachable,
	// and so could never be collected.
	for idx := len(kept); idx < len(q.nodes); idx++ {
		q.nodes[idx] = pnode[K, V]{}
	}

	q.nodes = kept

	q.heapify()
}

// heapify restores the heap property over the whole queue in O(n), and with it every
// entry's record of its own position.
func (q *pqueue[K, V]) heapify() {
	for idx := range q.nodes {
		q.nodes[idx].item.idx = idx
	}

	for idx := (len(q.nodes) / 2) - 1; idx >= 0; idx-- {
		q.down(idx)
	}
}

// top returns the entry with the earliest deadline, and whether there is one.
func (q *pqueue[K, V]) top() (K, *entry[V], bool) {
	if len(q.nodes) == 0 {
		var zero K

		return zero, nil, false
	}

	return q.nodes[0].key, q.nodes[0].item, true
}

// push queues the entry under the given key.
func (q *pqueue[K, V]) push(key K, item *entry[V]) {
	q.nodes = append(q.nodes, pnode[K, V]{item: item, key: key})
	item.idx = len(q.nodes) - 1

	q.up(item.idx)
}

// remove takes the entry out of the queue, wherever it sits in it: the hole left by an
// entry in the middle is filled with the last node, which is then sifted back into
// place.
func (q *pqueue[K, V]) remove(item *entry[V]) {
	idx, last := item.idx, len(q.nodes)-1

	if idx != last {
		q.place(idx, q.nodes[last])
	}

	q.nodes[last] = pnode[K, V]{} // clear the vacated slot, so the entry can be collected
	q.nodes = q.nodes[:last]

	if idx != last && !q.down(idx) {
		q.up(idx)
	}
}

// place moves the node to the given position, keeping the entry's record of it.
func (q *pqueue[K, V]) place(idx int, node pnode[K, V]) {
	q.nodes[idx] = node
	node.item.idx = idx
}

// swap exchanges two nodes, keeping their records of their positions.
func (q *pqueue[K, V]) swap(i, j int) {
	q.nodes[i], q.nodes[j] = q.nodes[j], q.nodes[i]
	q.nodes[i].item.idx, q.nodes[j].item.idx = i, j
}

// up sifts the node at idx towards the head until its parent is no later than it.
func (q *pqueue[K, V]) up(idx int) {
	for idx > 0 {
		parent := (idx - 1) / 2

		if !q.nodes[idx].item.deadline().Before(q.nodes[parent].item.deadline()) {
			return
		}

		q.swap(idx, parent)

		idx = parent
	}
}

// down sifts the node at idx away from the head, and reports whether it moved.
func (q *pqueue[K, V]) down(idx int) bool {
	start := idx

	for {
		first, right := (2*idx)+1, (2*idx)+2

		if first >= len(q.nodes) {
			break
		}

		if (right < len(q.nodes)) && q.nodes[right].item.deadline().Before(q.nodes[first].item.deadline()) {
			first = right
		}

		if !q.nodes[first].item.deadline().Before(q.nodes[idx].item.deadline()) {
			break
		}

		q.swap(idx, first)

		idx = first
	}

	return idx > start
}
