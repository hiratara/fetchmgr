package fetchmgr_test

import (
	"fmt"
	"math/rand"
	"strconv"
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
	benchmarkFetcher(b, func(fetcher SimpleFetcher) Fetcher {
		cf := NewCachedFetcher(
			MakeCancelable{fetcher},
			time.Second*10,
			5*time.Millisecond,
		)
		return MakeSimple{cf}
	})
}

func BenchmarkFetcher(b *testing.B) {
	benchmarkFetcher(b, func(fetcher SimpleFetcher) Fetcher {
		return New(
			MakeCancelable{fetcher},
			SetTTL(time.Second*10),
		)
	})
}

func benchmarkFetcher(b *testing.B, wrap func(SimpleFetcher) Fetcher) {
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
				for i := 0; i < n; i++ {
					rnd := rand.Intn(keynum)
					var key interface{}
					switch rnd % 3 {
					case 0:
						key = rnd
					case 1:
						key = float64(rnd)
					case 2:
						key = strconv.Itoa(rnd)
					}
					val, err := cached.Fetch(key)
					if err != nil || key != val {
						fmt.Printf("ERRRO: %v != %v, %v\r", key, val, err)
					}
				}
				wg.Done()
			}()
		}
		wg.Wait()

		fc, ok := cached.(FetchCloser)
		if ok {
			fc.Close()
		}

		if int(result) != keynum {
			fmt.Printf("Access to resource %d times, wants %d\r", result, keynum)
		}
	}
	fmt.Println("")
}
