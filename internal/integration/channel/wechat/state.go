package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

const (
	weixinDefaultCDNBaseURL    = "https://novac2c.cdn.weixin.qq.com/c2c"
	weixinConfigCacheTTL       = 24 * time.Hour
	weixinConfigRetryInitial   = 2 * time.Second
	weixinConfigRetryMax       = time.Hour
	weixinSessionPauseDuration = time.Hour
	weixinSessionExpiredCode   = -14
)

type typingTicketCacheEntry struct {
	ticket      string
	nextFetchAt time.Time
	retryDelay  time.Duration
}

type syncCursorFile struct {
	GetUpdatesBuf string `json:"get_updates_buf"`
}

type contextTokensFile struct {
	Tokens map[string]string `json:"tokens"`
}

func dataDir() string {
	return filepath.Join(".", "data", "wechat")
}

func syncCursorPath(cfg *config.WechatChannelConfig) string {
	dir := filepath.Join(dataDir(), "sync")
	return filepath.Join(dir, "cursor.json")
}

func contextTokensPath(cfg *config.WechatChannelConfig) string {
	dir := filepath.Join(dataDir(), "context-tokens")
	return filepath.Join(dir, "tokens.json")
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0o755)
}

func loadGetUpdatesBuf(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var decoded syncCursorFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		return "", err
	}
	return decoded.GetUpdatesBuf, nil
}

func saveGetUpdatesBuf(path, cursor string) error {
	data, err := json.Marshal(syncCursorFile{GetUpdatesBuf: cursor})
	if err != nil {
		return err
	}
	if err := ensureDir(path); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func loadContextTokens(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var decoded contextTokensFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	return decoded.Tokens, nil
}

func saveContextTokens(path string, tokens map[string]string) error {
	data, err := json.Marshal(contextTokensFile{Tokens: tokens})
	if err != nil {
		return err
	}
	if err := ensureDir(path); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func isSessionExpiredStatus(ret, errcode int) bool {
	return ret == weixinSessionExpiredCode || errcode == weixinSessionExpiredCode
}

func (c *WechatChannel) cdnBaseURL() string {
	if base := strings.TrimSpace(c.cfg.CDNBaseURL); base != "" {
		return strings.TrimRight(base, "/")
	}
	return weixinDefaultCDNBaseURL
}

func (c *WechatChannel) pauseSession(operation string, ret, errcode int, errmsg string) time.Duration {
	c.pauseMu.Lock()
	defer c.pauseMu.Unlock()

	until := time.Now().Add(weixinSessionPauseDuration)
	if until.After(c.pauseUntil) {
		c.pauseUntil = until
	}

	remaining := time.Until(c.pauseUntil)
	logger.ErrorCF("wechat", "Session expired; pausing WeChat channel", map[string]any{
		"operation": operation,
		"ret":       ret,
		"errcode":   errcode,
		"errmsg":    errmsg,
		"until":     c.pauseUntil.Format(time.RFC3339),
		"minutes":   int((remaining + time.Minute - 1) / time.Minute),
	})
	return remaining
}

func (c *WechatChannel) remainingPause() time.Duration {
	c.pauseMu.Lock()
	defer c.pauseMu.Unlock()

	if c.pauseUntil.IsZero() {
		return 0
	}
	remaining := time.Until(c.pauseUntil)
	if remaining <= 0 {
		c.pauseUntil = time.Time{}
		return 0
	}
	return remaining
}

func (c *WechatChannel) waitWhileSessionPaused(ctx context.Context) error {
	remaining := c.remainingPause()
	if remaining <= 0 {
		return nil
	}

	timer := time.NewTimer(remaining)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *WechatChannel) ensureSessionActive() error {
	remaining := c.remainingPause()
	if remaining <= 0 {
		return nil
	}
	return fmt.Errorf("wechat session paused (%d min remaining)", int((remaining+time.Minute-1)/time.Minute))
}

func (c *WechatChannel) getTypingTicket(ctx context.Context, userID string) (string, error) {
	now := time.Now()

	c.typingMu.Lock()
	entry, ok := c.typingCache[userID]
	if ok && now.Before(entry.nextFetchAt) {
		ticket := entry.ticket
		c.typingMu.Unlock()
		return ticket, nil
	}
	cachedTicket := entry.ticket
	retryDelay := entry.retryDelay
	c.typingMu.Unlock()

	contextToken := ""
	if v, ok := c.contextTokens.Load(userID); ok {
		contextToken, _ = v.(string)
	}

	resp, err := c.api.GetConfig(ctx, GetConfigReq{
		IlinkUserID:  userID,
		ContextToken: contextToken,
	})
	if err == nil && resp != nil && resp.Ret == 0 && resp.Errcode == 0 {
		ticket := strings.TrimSpace(resp.TypingTicket)
		c.typingMu.Lock()
		c.typingCache[userID] = typingTicketCacheEntry{
			ticket:      ticket,
			nextFetchAt: now.Add(weixinConfigCacheTTL),
			retryDelay:  weixinConfigRetryInitial,
		}
		c.typingMu.Unlock()
		return ticket, nil
	}

	if resp != nil && isSessionExpiredStatus(resp.Ret, resp.Errcode) {
		c.pauseSession("getconfig", resp.Ret, resp.Errcode, resp.Errmsg)
	}

	if retryDelay <= 0 {
		retryDelay = weixinConfigRetryInitial
	} else {
		retryDelay *= 2
		if retryDelay > weixinConfigRetryMax {
			retryDelay = weixinConfigRetryMax
		}
	}

	c.typingMu.Lock()
	c.typingCache[userID] = typingTicketCacheEntry{
		ticket:      cachedTicket,
		nextFetchAt: now.Add(retryDelay),
		retryDelay:  retryDelay,
	}
	c.typingMu.Unlock()

	if err != nil {
		return cachedTicket, err
	}
	if resp == nil {
		return cachedTicket, fmt.Errorf("getconfig returned nil response")
	}
	return cachedTicket, fmt.Errorf("getconfig failed: ret=%d errcode=%d errmsg=%s", resp.Ret, resp.Errcode, resp.Errmsg)
}

// WechatStateManager manages persistent state for the WeChat channel.
type WechatStateManager struct {
	cfg              *config.WechatChannelConfig
	mu               sync.Mutex
	contextTokens    map[string]string
	syncCursorPath   string
	contextTokenPath string
}

// NewWechatStateManager creates a state manager for the WeChat channel.
func NewWechatStateManager(cfg *config.WechatChannelConfig) *WechatStateManager {
	return &WechatStateManager{
		cfg:              cfg,
		contextTokens:    make(map[string]string),
		syncCursorPath:   syncCursorPath(cfg),
		contextTokenPath: contextTokensPath(cfg),
	}
}

// LoadCursor loads the persisted sync cursor.
func (sm *WechatStateManager) LoadCursor() (string, error) {
	return loadGetUpdatesBuf(sm.syncCursorPath)
}

// SaveCursor persists the sync cursor.
func (sm *WechatStateManager) SaveCursor(cursor string) error {
	return saveGetUpdatesBuf(sm.syncCursorPath, cursor)
}

// LoadContextTokens loads all persisted context tokens.
func (sm *WechatStateManager) LoadContextTokens() (map[string]string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	tokens, err := loadContextTokens(sm.contextTokenPath)
	if err != nil {
		return nil, err
	}
	if tokens != nil {
		sm.contextTokens = tokens
	}
	return tokens, nil
}

// GetContextToken returns the context token for a user.
func (sm *WechatStateManager) GetContextToken(userID string) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.contextTokens[userID]
}

// SetContextToken stores a context token for a user and persists to disk.
func (sm *WechatStateManager) SetContextToken(userID, token string) error {
	sm.mu.Lock()
	sm.contextTokens[userID] = token
	sm.mu.Unlock()
	return saveContextTokens(sm.contextTokenPath, sm.contextTokens)
}
