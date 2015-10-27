package fetchmgr

import (
	"hash/fnv"
	"sync"
	"time"
	"unsafe"
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
	bucketNum uint
	mutex     []sync.Mutex
	cache     []map[interface{}]entry
}

type entry struct {
	value   interface{}
	expires time.Time
}

// New creates CachedFetcher
func New(
	fetcher Fetcher,
	ss ...Setting,
) *CachedFetcher {
	cached := &CachedFetcher{
		fetcher:   fetcher,
		ttl:       1 * time.Minute,
		bucketNum: 10,
	}

	for _, set := range ss {
		set(cached)
	}

	cached.mutex = make([]sync.Mutex, cached.bucketNum)
	cached.cache = make([]map[interface{}]entry, cached.bucketNum)

	cached.prepare()

	return cached
}

// Setting makes arguments for New constracter
type Setting func(*CachedFetcher)

// SetTTL sets the expiration time of caches
func SetTTL(t time.Duration) Setting {
	return func(cf *CachedFetcher) {
		cf.ttl = t
	}
}

// SetBucketNum sets the number of map instance
// The default values is 10.
// Only Hasher instance, int, float64, string types support multiple map
// instance feature. If you don't use these types as keys, you had better
// set 1 for BucketNum.
func SetBucketNum(n uint) Setting {
	return func(cf *CachedFetcher) {
		cf.bucketNum = n
	}
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

	// TODO: Remove expired entries

	cached, ok := c.cache[h][key]
	if ok && time.Now().Before(cached.expires) {
		return cached.value, nil
	}

	val, err := c.fetcher.Fetch(key)
	if err != nil {
		return val, err
	}

	expires := time.Now().Add(c.ttl)
	c.cache[h][key] = entry{value: val, expires: expires}
	return val, nil
}

func (c *CachedFetcher) prepare() {
	for i := range c.cache {
		c.cache[i] = map[interface{}]entry{}
	}
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

// KFloat64 is hashable float64
type KFloat64 float64

// Hash calculates hash values
func (k KFloat64) Hash() uint {
	b := *(*[unsafe.Sizeof(k)]byte)(unsafe.Pointer(&k))

	h := fnv.New64a()
	h.Write(b[:])

	return uint(h.Sum64())
}

// KStr is hashable string
type KStr string

// Hash calculates hash values
func (k KStr) Hash() uint {
	h := fnv.New64a()
	h.Write([]byte(k))

	return uint(h.Sum64())
}
