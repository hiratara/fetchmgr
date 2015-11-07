package fetchmgr

import (
	"container/heap"
	"sync"
	"time"
)

// CachedFetcher caches fetched contents. It use Fetcher internally to fetch
// resources. It will call Fetcher's Fetch method thread-safely.
type CachedFetcher struct {
	fetcher  Fetcher
	ttl      time.Duration
	interval time.Duration
	mutex    sync.Mutex
	cache    map[interface{}]entry
	queMutex sync.Mutex
	queue    deleteQueue
	awake    chan struct{}
	closed   chan struct{}
}

type entry struct {
	value func() (interface{}, error)
}

// NewCachedFetcher creates CachedFetcher
func NewCachedFetcher(
	fetcher Fetcher,
	ttl time.Duration,
	interval time.Duration,
) *CachedFetcher {
	cached := &CachedFetcher{
		fetcher:  fetcher,
		ttl:      ttl,
		interval: interval,
		cache:    make(map[interface{}]entry),
		awake:    make(chan struct{}, 1),
		closed:   make(chan struct{}),
	}

	go deleteLoop(cached)

	return cached
}

// Fetch memoizes fetcher.Fetch method.
// It calls fetcher.Fetch method and caches the return value unless there is no
// cached results. Chached values are expired when c.ttl has passed.
// If the internal Fetcher.Fetch returns err (!= nil), CachedFetcher doesn't
// cache any results.
func (c *CachedFetcher) Fetch(key interface{}) (interface{}, error) {
	e := pickEntry(c, key)
	return e.value()
}

// Close closes this instance
func (c *CachedFetcher) Close() error {
	close(c.closed)

	fc, ok := c.fetcher.(FetchCloser)
	if ok {
		err := fc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func pickEntry(c *CachedFetcher, key interface{}) entry {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cached, ok := c.cache[key]
	if ok {
		return cached
	}

	var val interface{}
	var err error
	done := make(chan struct{})
	go func() {
		val, err = c.fetcher.Fetch(key)
		close(done)

		if err != nil {
			// Don't reuse error values
			deleteKeys(c, key)
			return
		}

		queueKey(c, key, c.ttl)
		awakeLoop(c)
	}()

	lazy := func() (interface{}, error) {
		<-done
		return val, err
	}

	cached = entry{value: lazy}
	c.cache[key] = cached

	return cached
}

func deleteKeys(c *CachedFetcher, keys ...interface{}) {
	if len(keys) == 0 {
		return // Lock nothing
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, k := range keys {
		delete(c.cache, k)
	}
}

func queueKey(c *CachedFetcher, key interface{}, ttl time.Duration) {
	c.queMutex.Lock()
	defer c.queMutex.Unlock()

	item := deleteItem{key, time.Now().Add(ttl)}
	heap.Push(&c.queue, item)
}

func deleteLoop(c *CachedFetcher) {
Loop:
	for {
		willDelete := make([]interface{}, 0, 1) // Will delete a few keys

		c.queMutex.Lock()
		for c.queue.Len() > 0 {
			item := heap.Pop(&c.queue).(deleteItem)
			if item.expire.After(time.Now()) {
				untilNext := item.expire.Sub(time.Now())
				heap.Push(&c.queue, item)
				go func() {
					time.Sleep(untilNext)
					awakeLoop(c)
				}()
				break
			}
			willDelete = append(willDelete, item.key)
		}
		c.queMutex.Unlock()

		// Delete here to avoid a dead lock
		deleteKeys(c, willDelete...)

		time.Sleep(c.interval) // Sleep a specified interval at least
		select {
		case <-c.closed:
			break Loop
		case <-c.awake:
		}
	}
}

func awakeLoop(c *CachedFetcher) {
	select {
	case c.awake <- struct{}{}:
	default:
	}
}

type deleteItem struct {
	key    interface{}
	expire time.Time
}

type deleteQueue []deleteItem

func (dq deleteQueue) Len() int { return len(dq) }

func (dq deleteQueue) Less(i, j int) bool {
	return dq[i].expire.Before(dq[j].expire)
}

func (dq deleteQueue) Swap(i, j int) {
	dq[i], dq[j] = dq[j], dq[i]
}

func (dq *deleteQueue) Push(x interface{}) {
	*dq = append(*dq, x.(deleteItem))
}

func (dq *deleteQueue) Pop() interface{} {
	n := len(*dq)
	ret := (*dq)[n-1]
	*dq = (*dq)[0 : n-1]
	return ret
}
