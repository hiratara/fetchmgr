package fetchmgr_test

import (
	"testing"
	"time"

	. "github.com/hiratara/fetchmgr"
)

type SlowFetcher struct{}

func (SlowFetcher) Fetch(key interface{}) (interface{}, error) {
	time.Sleep(time.Hour)
	return "Good morning", nil
}

func TestClose(t *testing.T) {
	var f SlowFetcher
	ccf := NewCachedFetcher(MakeCancelable{f}, time.Minute, time.Second)
	cf := MakeSimple{ccf}

	go func() {
		time.Sleep(time.Millisecond)
		ccf.Close()
	}()

	v, err := cf.Fetch("greeting")
	if err != ErrFetcherClosed {
		t.Fatalf("Gets (%v, %v), wants ErrFetcherClosed", v, err)
	}
}
