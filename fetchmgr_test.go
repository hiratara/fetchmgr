package fetchmgr_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/hiratara/fetchmgr"
)

type mapFetcher map[int]string

func (m mapFetcher) Fetch(key interface{}) (interface{}, error) {
	return m[key.(int)], nil
}

func TestCachedFetcher(t *testing.T) {
	fetcher := mapFetcher{1: "one", 2: "two"}
	cached := New(fetcher, time.Millisecond*10)

	one, err := str(cached.Fetch(1))
	if err != nil {
		t.Fatalf("Get error %v, wants nil", err)
	}
	if one != "one" {
		t.Fatalf(`Get %v, wants "one"`, one)
	}

	// Change values
	delete(fetcher, 1)
	one, err = str(cached.Fetch(1))
	if err != nil {
		t.Fatalf("Get error %v, wants nil", err)
	}
	if one != "one" {
		t.Fatalf(`Get %v, wants "one" (cached data)`, one)
	}

	// Waiting for clearing caches
	time.Sleep(time.Millisecond * 20)
	one, err = str(cached.Fetch(1))
	if err != nil {
		t.Fatalf("Get error %v, wants nil", err)
	}
	if one != "" {
		t.Fatalf("Get %v, wants nil", one)
	}

	// Change values
	fetcher[1] = "ONE"
	one, err = str(cached.Fetch(1))
	if err != nil {
		t.Fatalf("Get error %v, wants nil", err)
	}
	if one != "" {
		t.Fatalf(`Get %v, wants nil (cached data)`, one)
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
