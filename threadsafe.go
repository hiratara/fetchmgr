package fetchmgr

import (
	"io"
	"sync"
)

// SafeFetcher is a synced instance of Fetcher
type safeCFetcher struct {
	mutex   *sync.Mutex
	fetcher CFetcher
}

// NewSafeCFetcher makes f thread-safe. It will be a slow instance because
// all CFetch() calls are serialized.
func NewSafeCFetcher(f CFetcher) CFetcher {
	return newSafeCFetcher(f)
}

func newSafeCFetcher(f CFetcher) safeCFetcher {
	var mutex sync.Mutex
	return safeCFetcher{&mutex, f}
}

// CFetch fetches a value
func (sf safeCFetcher) CFetch(cancel <-chan struct{}, k interface{}) (interface{}, error) {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()
	return sf.fetcher.CFetch(cancel, k)
}

// safeCFetchCloser a synced instance of FetchCloser
type safeCFetchCloser struct {
	safeCFetcher
	io.Closer
}

// NewSafeCFetchCloser makes fc thread-safe. It will be a slow instance
// because all CFetch() and Close() calls are serialized.
func NewSafeCFetchCloser(fc CFetchCloser) CFetchCloser {
	sf := newSafeCFetcher(fc)
	return safeCFetchCloser{sf, fc}
}

// Close closes sfc
func (sfc safeCFetchCloser) Close() error {
	m := sfc.safeCFetcher.mutex
	m.Lock()
	defer m.Unlock()
	return sfc.Closer.Close()
}

// NewSafeFetcher makes f thread-safe. It will be a slow instance because
// all Fetch() calls are serialized.
func NewSafeFetcher(f Fetcher) Fetcher {
	cf := AsCFetcher{f}
	sfcf := NewSafeCFetcher(cf)
	return AsFetcher{sfcf}
}

// NewSafeFetchCloser makes fc thread-safe. It will be a slow instance
// because all Fetch() and Close() calls are serialized.
func NewSafeFetchCloser(fc FetchCloser) FetchCloser {
	cfc := struct {
		CFetcher
		io.Closer
	}{AsCFetcher{fc}, fc}
	sfcfc := NewSafeCFetchCloser(cfc)
	return struct {
		Fetcher
		io.Closer
	}{AsFetcher{sfcfc}, sfcfc}
}
