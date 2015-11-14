package fetchmgr

import (
	"errors"
	"io"
	"time"
)

// SimpleFetcher is the interface in order to fetch outer resources
type SimpleFetcher interface {
	Fetch(interface{}) (interface{}, error)
}

// CancelableFetcher is the interface in order to fetch outer resources
// It also provides a cancel chan to cancel fetching.
type CancelableFetcher interface {
	CancelableFetch(chan struct{}, interface{}) (interface{}, error)
}

// ErrFetchCanceled means the CancelableFetch call was canceled
var ErrFetchCanceled = errors.New("calling Fetch canceled")

// Fetcher is the interface in order to fetch outer resources
type Fetcher interface {
	SimpleFetcher
	CancelableFetcher
}

// FetchCloser has Fetch and Close method
type FetchCloser interface {
	Fetcher
	io.Closer
}

// MakeCancelable makes Fetcher from SimpleFetcher
type MakeCancelable struct {
	SimpleFetcher
}

// CancelableFetch fetches resources and provides the cancel chan
func (tf MakeCancelable) CancelableFetch(cancel chan struct{}, key interface{}) (interface{}, error) {
	return tf.Fetch(key)
}

// MakeSimple makes Fetcher from CancelableFetcher
type MakeSimple struct {
	CancelableFetcher
}

// Fetch fetches resources
func (tf MakeSimple) Fetch(key interface{}) (interface{}, error) {
	return tf.CancelableFetch(nil, key)
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

	fs := make([]CancelableFetcher, setting.bucketNum)
	for i := range fs {
		fs[i] = NewCachedFetcher(fetcher, setting.ttl, setting.interval)
	}

	nbf := NewBucketedFetcher(fs)
	return struct {
		BucketedFetcher
		SimpleFetcher
	}{nbf, MakeSimple{nbf}}
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
