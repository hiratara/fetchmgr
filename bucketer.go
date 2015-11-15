package fetchmgr

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"io"
	"unsafe"
)

// BucketedCFetcher holds multiple fetchers and scatters tasks
// by hash values of keys
type BucketedCFetcher []CFetcher

// NewBucketedCFetcher creates the instance
func NewBucketedCFetcher(fs []CFetcher) BucketedCFetcher {
	return BucketedCFetcher(fs)
}

// CFetch calls one of internal Fetchers
func (bf BucketedCFetcher) CFetch(cancel chan struct{}, key interface{}) (interface{}, error) {
	fs := ([]CFetcher)(bf)
	i := hash(key) % uint(len(fs))
	return fs[i].CFetch(cancel, key)
}

// InnerError has been occured in internal Fetcher()
type InnerError struct {
	Fetcher CFetcher
	Err     error
}

func (ie InnerError) Error() string {
	return fmt.Sprintf("%v: %v", ie.Fetcher, ie.Err)
}

// InnerErrors is a list of InnerError
type InnerErrors []InnerError

func (ies InnerErrors) Error() string {
	var buf bytes.Buffer
	for _, ie := range ies {
		buf.WriteString(ie.Error() + "\n")
	}
	return buf.String()
}

// Close calls Close() for all internal FetchCloser instances
func (bf BucketedCFetcher) Close() error {
	var errs []InnerError
	for _, f := range bf {
		switch ff := f.(type) {
		case io.Closer:
			err := ff.Close()
			errs = append(errs, InnerError{f, err})
		}
	}
	if len(errs) > 0 {
		return InnerErrors(errs)
	}

	return nil
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
