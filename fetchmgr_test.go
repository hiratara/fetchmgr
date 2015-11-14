package fetchmgr_test

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/hiratara/fetchmgr"
)

type fetchRet struct {
	value string
	err   error
}

type mapFetcher map[int]fetchRet

func (m mapFetcher) Fetch(key interface{}) (interface{}, error) {
	r := m[key.(int)]
	return r.value, r.err
}

func TestCachedFetcher(t *testing.T) {
	fetcher := mapFetcher{
		1: {"one", nil},
		2: {"two", nil},
		3: {"", errors.New("no 3rd elems")},
	}
	cached := New(
		MakeCancelable{fetcher},
		SetInterval(time.Millisecond*1),
		SetTTL(time.Millisecond*100),
	)
	defer cached.Close()

	one, err := str(cached.Fetch(1))
	if err != nil {
		t.Fatalf("Get error %v, wants nil", err)
	}
	if one != "one" {
		t.Fatalf(`Get %v, wants "one"`, one)
	}

	three, err := str(cached.Fetch(3))
	if err == nil {
		t.Fatal(`"Gets nil, wants "no 3rd elems" err`)
	}
	if three != "" {
		t.Fatalf(`Gets %v, wants ""`, three)
	}

	// Then, fetch another values
	time.Sleep(time.Millisecond * 55)
	two, err := str(cached.Fetch(2))
	if err != nil {
		t.Fatalf("Get error %v, wants nil", err)
	}
	if two != "two" {
		t.Fatalf(`Get %v, wants "two"`, two)
	}

	// Change values
	delete(fetcher, 1)
	delete(fetcher, 2)
	fetcher[3] = fetchRet{"three", nil}
	one, err = str(cached.Fetch(1))
	if err != nil {
		t.Fatalf("Get error %v, wants nil", err)
	}
	if one != "one" {
		t.Fatalf(`Get %v, wants "one" (cached data)`, one)
	}

	three, err = str(cached.Fetch(3))
	if err != nil {
		t.Fatalf("Get error %v, wants nil", err)
	}
	if three != "three" {
		t.Fatalf(`Gets %v, wants "three" (fetched normaly)`, one)
	}

	// Waiting for clearing caches for "one"
	time.Sleep(time.Millisecond * 55)
	one, err = str(cached.Fetch(1))
	if err != nil {
		t.Fatalf("Get error %v, wants nil", err)
	}
	if one != "" {
		t.Fatalf("Get %v, wants nil", one)
	}

	two, err = str(cached.Fetch(2))
	if err != nil {
		t.Fatalf("Get error %v, wants nil", err)
	}
	if two != "two" {
		t.Fatalf(`Get %v, wants "two" (cached data)`, two)
	}

	// Change values
	fetcher[1] = fetchRet{"ONE", nil}
	fetcher[2] = fetchRet{"TWO", nil}
	one, err = str(cached.Fetch(1))
	if err != nil {
		t.Fatalf("Get error %v, wants nil", err)
	}
	if one != "" {
		t.Fatalf(`Get %v, wants nil (cached data)`, one)
	}

	// Waiting for clearing caches for "two"
	time.Sleep(time.Millisecond * 55)
	two, err = str(cached.Fetch(2))
	if err != nil {
		t.Fatalf("Get error %v, wants nil", err)
	}
	if two != "TWO" {
		t.Fatalf(`Get %v, wants "TWO"`, two)
	}
}

func str(v interface{}, err error) (string, error) {
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("Not a string: %v", v)
	}
	return s, nil
}

type constFetcher int

func (c constFetcher) Fetch(key interface{}) (interface{}, error) {
	return "const", nil
}

func TestCachedFetcherNan(t *testing.T) {
	cached := New(MakeCancelable{constFetcher(0)})
	defer cached.Close()

	ks := []float64{math.NaN(), math.Inf(+1), math.Inf(-1)}

	for _, k := range ks {
		v, err := cached.Fetch(k)
		if err != nil {
			t.Fatalf("Shouldn't throw errors for %.f: %v", k, err)
		}
		if v != "const" {
			t.Fatalf(`Gets %v for %.f, wants "const"`, k, v)
		}
	}
}

type testCFetcher struct {
	wg  sync.WaitGroup
	cnt uint32
}

func (cf *testCFetcher) CancelableFetch(cancel chan struct{}, key interface{}) (interface{}, error) {
	cf.wg.Add(1)
	select {
	case <-cancel:
		atomic.AddUint32(&cf.cnt, 1)
		cf.wg.Done()
		return nil, errors.New("canceled")
	}
}

func TestCancelAndClose(t *testing.T) {
	cf := &testCFetcher{}
	ccf := New(MakeSimple{cf})

	var canceled uint32

	done1 := make(chan struct{})
	cancel1 := make(chan struct{})
	go func() {
		_, err := ccf.CancelableFetch(cancel1, "key")
		if err == nil {
			t.Fatalf("Gets nil, wants errors")
		}
		atomic.AddUint32(&canceled, 1)
		close(done1)
	}()

	done2 := make(chan struct{})
	go func() {
		_, err := ccf.CancelableFetch(nil, "key")
		if err == nil {
			t.Fatalf("Gets nil, wants errors")
		}
		atomic.AddUint32(&canceled, 1)
		close(done2)
	}()

	done3 := make(chan struct{})
	go func() {
		_, err := ccf.CancelableFetch(nil, "KEY")
		if err == nil {
			t.Fatalf("Gets nil, wants errors")
		}
		atomic.AddUint32(&canceled, 1)
		close(done3)
	}()

	close(cancel1)
	<-done1
	time.Sleep(10 * time.Millisecond) // Check if done2 isn't canceled
	if canceled != 1 {
		t.Fatalf("Gets %d canceled, wants 1", canceled)
	}
	if cf.cnt != 0 {
		t.Fatalf("Gets %d canceled internal calls, wants 0", canceled)
	}

	ccf.Close()
	<-done2
	<-done3
	cf.wg.Wait()
	time.Sleep(10 * time.Millisecond) // Wait for all cancel calls
	if canceled != 3 {
		t.Fatalf("Gets %d canceled, wants 3", canceled)
	}
	if cf.cnt != 2 {
		t.Fatalf(`Gets %d canceled internal calls, wants 2 ("key" and "KEY")`, cf.cnt)
	}
}
