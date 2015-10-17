package fetchmgr_test

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/hiratara/fetchmgr"
)

var fetchnum = 10000
var conc = 100
var keynum = 1000

type slowIdentityFetcher int32

func (cnt *slowIdentityFetcher) Fetch(key interface{}) (interface{}, error) {
	time.Sleep(1 * time.Millisecond)
	atomic.AddInt32((*int32)(cnt), 1)
	return key, nil
}

func BenchmarkCachedFetcher(b *testing.B) {
	benchmarkFetcher(b, func(fetcher Fetcher) Fetcher {
		return New(fetcher, time.Second*10)
	})
}

func benchmarkFetcher(b *testing.B, wrap func(Fetcher) Fetcher) {
	b.StopTimer()
	var baseN = fetchnum / conc
	b.StartTimer()

	for j := 0; j < b.N; j++ {
		b.StopTimer()
		var result slowIdentityFetcher
		cached := wrap(&result)
		b.StartTimer()

		left := fetchnum
		var wg sync.WaitGroup
		wg.Add(conc)
		for i := 0; i < conc; i++ {
			var n int
			if i == conc-1 {
				n = left
			} else {
				n = baseN
			}
			left -= n
			go func() {
				for k := 0; k < n; k++ {
					k := rand.Intn(keynum)
					v, err := cached.Fetch(k)
					if err != nil || k != v {
						fmt.Printf("ERRRO: %d != %d, %v\r", k, v, err)
					}
				}
				wg.Done()
			}()
		}
		wg.Wait()

		if int(result) != keynum {
			fmt.Printf("Access to resource %d times, wants %d", result, keynum)
		}
	}
}
