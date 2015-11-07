package fetchmgr

import (
	"io"
	"time"
)

// Fetcher is the interface in order to fetch outer resources
type Fetcher interface {
	Fetch(interface{}) (interface{}, error)
}

// FetchCloser has Fetch and Close method
type FetchCloser interface {
	Fetcher
	io.Closer
}

// FuncFetcher makes new Fetcher from a function
type FuncFetcher func(interface{}) (interface{}, error)

// Fetch calls the internal function
func (f FuncFetcher) Fetch(k interface{}) (interface{}, error) {
	return f(k)
}

// New wraps the fetcher and memoizes the results for Fetch
func New(
	fetcher Fetcher,
	ss ...Setting,
) FetchCloser {
	setting := &fetcherSetting{
		bucketNum: 10,
		ttl:       1 * time.Minute,
		interval:  1 * time.Second,
	}

	for _, set := range ss {
		set(setting)
	}

	fs := make([]Fetcher, setting.bucketNum)
	for i := range fs {
		fs[i] = NewCachedFetcher(fetcher, setting.ttl, setting.interval)
	}

	return NewBucketedFetcher(fs)
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
