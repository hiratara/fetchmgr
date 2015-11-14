package fetchmgr_test

import (
	"errors"
	"fmt"
	"math"
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
