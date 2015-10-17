package fetchmgr

import (
	"sync"
	"time"
)

// Fetcher is the interface in order to fetch outer resources
type Fetcher interface {
	Fetch(interface{}) (interface{}, error)
}

// FuncFetcher makes new Fetcher from a function
type FuncFetcher func(interface{}) (interface{}, error)

// Fetch calls the internal function
func (f FuncFetcher) Fetch(k interface{}) (interface{}, error) {
	return f(k)
}

// CachedFetcher caches fetched contents. It use Fetcher internally to fetch
// resources. It will call Fetcher's Fetch method thread-safely.
type CachedFetcher struct {
	fetcher Fetcher
	mutex   sync.Mutex
	ttl     time.Duration
	expires time.Time
	cache   map[interface{}]interface{}
}

// New creates CachedFetcher
func New(
	fetcher Fetcher,
	ttl time.Duration,
) *CachedFetcher {
	cache := &CachedFetcher{
		fetcher: fetcher,
		ttl:     ttl,
	}
	cache.prepare()

	return cache
}

// Fetch memoizes fetcher.Fetch method.
// It calls fetcher.Fetch method and caches the return value unless there is no
// cached results. Chached values are expired when c.ttl has passed.
// If the internal Fetcher.Fetch returns err (!= nil), CachedFetcher doesn't
// cache any results.
func (c *CachedFetcher) Fetch(key interface{}) (interface{}, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cleanup()

	cached, ok := c.cache[key]
	if ok {
		return cached, nil
	}

	val, err := c.fetcher.Fetch(key)
	if err != nil {
		return val, err
	}

	c.cache[key] = val
	return val, nil
}

func (c *CachedFetcher) prepare() {
	c.cache = map[interface{}]interface{}{}
	c.expires = time.Now().Add(c.ttl)
}

func (c *CachedFetcher) cleanup() {
	if time.Now().Before(c.expires) {
		return
	}

	c.prepare()
}
