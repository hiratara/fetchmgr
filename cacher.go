package fetchmgr

import (
	"sync"
	"time"
)

// CachedFetcher caches fetched contents. It use Fetcher internally to fetch
// resources. It will call Fetcher's Fetch method thread-safely.
type CachedFetcher struct {
	fetcher Fetcher
	ttl     time.Duration
	mutex   sync.Mutex
	cache   map[interface{}]entry
}

type entry struct {
	value func() (interface{}, error)
}

// NewCachedFetcher creates CachedFetcher
func NewCachedFetcher(
	fetcher Fetcher,
	ttl time.Duration,
) *CachedFetcher {
	cached := &CachedFetcher{
		fetcher: fetcher,
		ttl:     ttl,
		cache:   make(map[interface{}]entry),
	}

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
		defer c.deleteKey(key)

		val, err = c.fetcher.Fetch(key)
		close(done)

		if err != nil {
			// Don't reuse error values
			return
		}

		// Wait for our entry expired
		time.Sleep(c.ttl)
	}()

	lazy := func() (interface{}, error) {
		<-done
		return val, err
	}

	cached = entry{value: lazy}
	c.cache[key] = cached

	return cached
}

func (c *CachedFetcher) deleteKey(key interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.cache, key)
}

func hash(k interface{}) uint {
	switch kk := k.(type) {
	case Hasher:
		return kk.Hash()
	case int:
		kkk := KInt(kk)
		return kkk.Hash()
	case float64:
		kkk := KFloat64(kk)
		return kkk.Hash()
	case string:
		return KStr(kk).Hash()
	}
	return 0
}
