package ctxfetchmgr_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"

	. "github.com/hiratara/fetchmgr/ctxfetchmgr"
)

type slowFetcher struct{}

func (slowFetcher) CFetch(cancel <-chan struct{}, key interface{}) (interface{}, error) {
	t := time.NewTimer(100 * time.Millisecond)
	defer t.Stop()

	select {
	case <-t.C:
		return true, nil
	case <-cancel:
		return nil, errors.New("canceled")
	}
}

func TestCtxFetch(t *testing.T) {
	fetcher := CNew(slowFetcher{})
	defer fetcher.Close()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
		defer cancel()

		_, err := fetcher.CtxFetch(ctx, "key")
		if err != nil {
			t.Fatalf("Thrown %v, wants nil", err)
		}

		_, err = fetcher.CtxFetch(ctx, "key")
		if err != nil {
			t.Fatalf("Thrown %v, wants nil (from cache)", err)
		}

		v, err := fetcher.CtxFetch(ctx, "anotherKey") // !!timeout!!
		if err == nil {
			t.Fatalf("Successfully gets %v, wants timeout", v)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		ctx := context.Background()
		ctx, anotherCancel := context.WithCancel(ctx)
		defer anotherCancel()

		// Waiting the 1st goroutine to fetch "key"
		time.Sleep(50 * time.Millisecond)
		_, err := fetcher.CtxFetch(ctx, "key")
		if err != nil {
			t.Fatalf("Thrown %v, wants nil (from cache after timeout)", err)
		}

		// Waiting the 1st goroutine to fetch "anotherKey"
		time.Sleep(50 * time.Millisecond)
		_, err = fetcher.CtxFetch(ctx, "anotherKey")
		if err != nil {
			t.Fatalf("Thrown %v, wants nil (result of another goroutine)", err)
		}
	}()

	wg.Wait()
}
