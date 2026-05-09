package feishu

import (
	"context"
	"sync"
	"time"
)

// tokenCache implements larkcore.Cache with InvalidateAll.
//
// The Lark SDK's built-in retry does not clear stale tenant_access_token
// from its cache on auth error (code 99991663), causing all API calls to
// fail until the token naturally expires (~2 hours). This custom cache
// works around that by exposing InvalidateAll().
type tokenCache struct {
	mu    sync.RWMutex
	store map[string]*tokenEntry
}

type tokenEntry struct {
	value    string
	expireAt time.Time
}

func newTokenCache() *tokenCache {
	return &tokenCache{store: make(map[string]*tokenEntry)}
}

func (c *tokenCache) Set(_ context.Context, key, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = &tokenEntry{value: value, expireAt: time.Now().Add(ttl)}
	return nil
}

func (c *tokenCache) Get(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.store[key]
	if !ok {
		return "", nil
	}
	if e.expireAt.Before(time.Now()) {
		delete(c.store, key)
		return "", nil
	}
	return e.value, nil
}

// InvalidateAll removes all cached tokens, forcing fresh acquisition.
func (c *tokenCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.store)
}
