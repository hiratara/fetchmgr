package fetchmgr

import (
	"io"
	"sync"
)

// SafeFetcher is a synced instance of Fetcher
type SafeFetcher struct {
	mutex   *sync.Mutex
	fetcher Fetcher
}

// NewSafeFetcher makes f thread-safe. It will be a slow instance because
// all Fetch() calls are serialized.
func NewSafeFetcher(f Fetcher) Fetcher {
	return newSafeFetcher(f)
}

func newSafeFetcher(f Fetcher) SafeFetcher {
	var mutex sync.Mutex
	return SafeFetcher{&mutex, f}
}

// Fetch fetches a value
func (sf SafeFetcher) Fetch(k interface{}) (interface{}, error) {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()
	return sf.fetcher.Fetch(k)
}

// SafeFetchCloser a synced instance of FetchCloser
type SafeFetchCloser struct {
	SafeFetcher
	io.Closer
}

// NewSafeFetchCloser makes fc thread-safe. It will be a slow instance
// because all Fetch() calls are serialized.
func NewSafeFetchCloser(fc FetchCloser) FetchCloser {
	sf := newSafeFetcher(fc)
	return SafeFetchCloser{sf, fc}
}

// Close closes sfc
func (sfc SafeFetchCloser) Close() error {
	m := sfc.SafeFetcher.mutex
	m.Lock()
	defer m.Unlock()
	return sfc.Closer.Close()
}
