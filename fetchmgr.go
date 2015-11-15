package fetchmgr

import (
	"errors"
	"io"
	"time"
)

// CFetcher is the interface in order to fetch outer resources
// It also provides a cancel chan to cancel fetching.
type CFetcher interface {
	CFetch(chan struct{}, interface{}) (interface{}, error)
}

// ErrFetchCanceled means the CFetch call was canceled
var ErrFetchCanceled = errors.New("calling Fetch canceled")

// CFetchCloser has Fetch and Close method
type CFetchCloser interface {
	CFetcher
	io.Closer
}

// Fetcher is the interface in order to fetch outer resources
type Fetcher interface {
	Fetch(interface{}) (interface{}, error)
}

// FetchCloser has Fetch and Close method
type FetchCloser interface {
	Fetcher
	io.Closer
}

type asCFetcher struct {
	Fetcher
}

func (tf asCFetcher) CFetch(cancel chan struct{}, key interface{}) (interface{}, error) {
	return tf.Fetch(key)
}

type asFetcher struct {
	CFetcher
}

func (tf asFetcher) Fetch(key interface{}) (interface{}, error) {
	return tf.CFetch(nil, key)
}

// FuncFetcher makes new Fetcher from a function
type FuncFetcher func(interface{}) (interface{}, error)

// Fetch calls the internal function
func (f FuncFetcher) Fetch(k interface{}) (interface{}, error) {
	return f(k)
}

// CNew wraps the fetcher and memoizes the results for Fetch
func CNew(
	fetcher CFetcher,
	ss ...Setting,
) CFetchCloser {
	setting := &fetcherSetting{
		bucketNum: 10,
		ttl:       1 * time.Minute,
		interval:  1 * time.Second,
	}

	for _, set := range ss {
		set(setting)
	}

	fs := make([]CFetcher, setting.bucketNum)
	for i := range fs {
		fs[i] = NewCachedCFetcher(fetcher, setting.ttl, setting.interval)
	}

	return NewBucketedCFetcher(fs)
}

// New wraps the fetcher and memoizes the results for Fetch
func New(
	fetcher Fetcher,
	ss ...Setting,
) FetchCloser {
	cfetcher := asCFetcher{fetcher}
	ccfetcher := CNew(cfetcher, ss...)
	return struct {
		Fetcher
		io.Closer
	}{asFetcher{ccfetcher}, ccfetcher}
}

type fetcherSetting struct {
	ttl       time.Duration
	interval  time.Duration
	bucketNum uint
}

// Setting makes arguments for New constracter
type Setting func(*fetcherSetting)

// SetTTL sets the expiration time of caches
func SetTTL(t time.Duration) Setting {
	return func(cf *fetcherSetting) {
		cf.ttl = t
	}
}

// SetInterval sets an interval to check expirations
func SetInterval(t time.Duration) Setting {
	return func(cf *fetcherSetting) {
		cf.interval = t
	}
}

// SetBucketNum sets the number of map instance
// The default values is 10.
// Only Hasher instance, int, float64, string types support multiple map
// instance feature. If you don't use these types as keys, you had better
// set 1 for BucketNum.
func SetBucketNum(n uint) Setting {
	return func(cf *fetcherSetting) {
		cf.bucketNum = n
	}
}
