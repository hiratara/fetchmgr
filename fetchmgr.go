package fetchmgr

import (
	"hash/fnv"
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
	fetcher   Fetcher
	ttl       time.Duration
	expires   time.Time
	bucketNum uint
	mutex     []sync.Mutex
	cache     []map[interface{}]interface{}
}

// New creates CachedFetcher
func New(
	fetcher Fetcher,
	ttl time.Duration,
) *CachedFetcher {
	n := 10
	mutex := make([]sync.Mutex, n)
	cache := make([]map[interface{}]interface{}, n)

	cached := &CachedFetcher{
		fetcher:   fetcher,
		ttl:       ttl,
		bucketNum: uint(n),
		mutex:     mutex,
		cache:     cache,
	}
	cached.prepare()

	return cached
}

// Fetch memoizes fetcher.Fetch method.
// It calls fetcher.Fetch method and caches the return value unless there is no
// cached results. Chached values are expired when c.ttl has passed.
// If the internal Fetcher.Fetch returns err (!= nil), CachedFetcher doesn't
// cache any results.
func (c *CachedFetcher) Fetch(key interface{}) (interface{}, error) {
	h := hash(key) % c.bucketNum

	c.mutex[h].Lock()
	defer c.mutex[h].Unlock()

	c.cleanup()

	cached, ok := c.cache[h][key]
	if ok {
		return cached, nil
	}

	val, err := c.fetcher.Fetch(key)
	if err != nil {
		return val, err
	}

	c.cache[h][key] = val
	return val, nil
}

func (c *CachedFetcher) prepare() {
	for i := range c.cache {
		c.cache[i] = map[interface{}]interface{}{}
	}
	c.expires = time.Now().Add(c.ttl)
}

func (c *CachedFetcher) cleanup() {
	if time.Now().Before(c.expires) {
		return
	}

	c.prepare()
}

func hash(k interface{}) uint {
	switch kk := k.(type) {
	case Hasher:
		return kk.Hash()
	case int:
		kkk := KInt(kk)
		return kkk.Hash()
	case string:
		return KStr(kk).Hash()
	}
	return 0
}

// Hasher provides a function in order to calculate its hash values
type Hasher interface {
	Hash() uint
}

// KInt is hashable int
type KInt int

// Hash calculates hash values
func (k KInt) Hash() uint {
	return uint(k)
}

// KStr is hashable string
type KStr string

// Hash calculates hash values
func (k KStr) Hash() uint {
	h := fnv.New64a()
	h.Write([]byte(k))

	return uint(h.Sum64())
}
