package fetchmgr_test

import (
	"io"
	"sync"
	"testing"

	. "github.com/hiratara/fetchmgr"
)

type UnsafeFetcher int32

func (cnt *UnsafeFetcher) Fetch(key interface{}) (interface{}, error) {
	*cnt++
	return *cnt, nil
}

func (cnt *UnsafeFetcher) Close() error {
	*cnt = -*cnt
	return nil
}

func TestSafeFetcher(t *testing.T) {
	var f UnsafeFetcher
	sf := NewSafeFetcher(&f)

	fetch10000Times(t, sf)

	if n := int32(f); n != 10000 {
		t.Fatalf("Gets %d, wants 10000", n)
	}
}

func TestSafeFetchCloser(t *testing.T) {
	var f UnsafeFetcher
	sf := NewSafeFetchCloser(&f)

	fetch10000Times(t, sf)

	if n := int32(f); n != -10000 {
		t.Fatalf("Gets %d, wants -10000", n)
	}
}

func fetch10000Times(t *testing.T, f Fetcher) {
	c, ok := f.(io.Closer)
	if ok {
		defer c.Close()
	}

	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, _ = f.Fetch(nil)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
