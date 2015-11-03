package fetchmgr

import (
	"hash/fnv"
	"unsafe"
)

// BucketedFetcher holds multiple fetchers and scatters tasks
// by hash values of keys
type BucketedFetcher []Fetcher

// NewBucketedFetcher creates the instance
func NewBucketedFetcher(fs []Fetcher) BucketedFetcher {
	return BucketedFetcher(fs)
}

// Fetch calls one of internal Fetchers
func (bf BucketedFetcher) Fetch(key interface{}) (interface{}, error) {
	fs := ([]Fetcher)(bf)
	i := hash(key) % uint(len(fs))
	return fs[i].Fetch(key)
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
