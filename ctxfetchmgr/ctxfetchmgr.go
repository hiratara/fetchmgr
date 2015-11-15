// Package ctxfetchmgr provides context-aware fetchmgr
package ctxfetchmgr

import (
	"github.com/hiratara/fetchmgr"
	"golang.org/x/net/context"
)

// ContextFetcher is a context-aware Fetcher
type ContextFetcher struct {
	fetcher fetchmgr.CFetchCloser
}

// CNew makes the new ContextFetcher from CFetcher
func CNew(
	fetcher fetchmgr.CFetcher,
	ss ...fetchmgr.Setting,
) ContextFetcher {
	cached := fetchmgr.CNew(fetcher, ss...)
	return ContextFetcher{cached}
}

// Close closes underlying fetcher
func (f ContextFetcher) Close() error {
	return f.fetcher.Close()
}

// CtxFetch fetches values. You can cancel the task by using ctx.Done()
func (f ContextFetcher) CtxFetch(
	ctx context.Context,
	k interface{},
) (interface{}, error) {
	return f.fetcher.CFetch(ctx.Done(), k)
}

// New makes the new ContextFetcher from Fetcher.
// CtxFetch of this instance won't be canceled.
func New(
	fetcher fetchmgr.Fetcher,
	ss ...fetchmgr.Setting,
) ContextFetcher {
	cfetcher := fetchmgr.AsCFetcher{Fetcher: fetcher}
	return CNew(cfetcher, ss...)
}
