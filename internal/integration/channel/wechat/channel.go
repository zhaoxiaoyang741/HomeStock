package wechat

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/channel"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// WechatChannel implements channel.Channel for personal WeChat
// via the Tencent iLink REST API (same approach as picoclaw).
type WechatChannel struct {
	*channel.BaseChannel
	cfg           config.WechatChannelConfig
	api           *ApiClient
	ctx           context.Context
	cancel        context.CancelFunc
	stateMgr      *WechatStateManager
	contextTokens sync.Map // from_user_id → context_token (in-memory cache)
	notifyChatID  atomic.Value
	mu            sync.Mutex
	typingMu      sync.Mutex
	typingCache   map[string]typingTicketCacheEntry
	pauseMu       sync.Mutex
	pauseUntil                  time.Time
	syncCursor                  string
	syncCursorMu                sync.Mutex
	consecutiveSessionExpiries  int
	stopped                     chan struct{}
	stopOnce                    sync.Once
}

// NewWechatChannel creates a WechatChannel from the given config.
func NewWechatChannel(cfg config.WechatChannelConfig) *WechatChannel {
	c := &WechatChannel{
		BaseChannel: &channel.BaseChannel{},
		cfg:         cfg,
		typingCache: make(map[string]typingTicketCacheEntry),
		stateMgr:    NewWechatStateManager(&cfg),
		stopped:     make(chan struct{}),
	}
	c.InitBase("wechat", nil)
	return c
}

// Name returns the channel name.
func (c *WechatChannel) Name() string { return "wechat" }

// Start initializes the API client and begins the long-poll receive loop.
func (c *WechatChannel) Start(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.BaseStart(c.ctx)

	baseURL := c.cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://ilinkai.weixin.qq.com/"
	}

	api, err := NewApiClient(baseURL, c.cfg.Token, c.cfg.Proxy)
	if err != nil {
		c.cancel()
		c.BaseStop()
		return fmt.Errorf("wechat: failed to create API client: %w", err)
	}
	c.api = api

	// Restore persisted state
	if err := c.restoreState(); err != nil {
		logger.WarnCF("wechat", "Failed to restore state", map[string]any{"error": err.Error()})
	}

	// Start the polling loop
	go c.pollLoop(c.ctx)

	logger.InfoCF("wechat", "channel started", map[string]any{
		"base_url": baseURL,
	})
	return nil
}

// Stop gracefully stops the poll loop and cleans up.
func (c *WechatChannel) Stop(ctx context.Context) error {
	logger.InfoC("wechat", "stopping channel")

	if c.cancel != nil {
		c.cancel()
	}

	c.stopOnce.Do(func() {
		close(c.stopped)
	})

	c.BaseStop()
	logger.InfoC("wechat", "channel stopped")
	return nil
}

// NotifyChatID returns the ChatID of the last user who sent a message.
// Implements channel.NotifyTargetProvider for system notifications.
func (c *WechatChannel) NotifyChatID() string {
	id, _ := c.notifyChatID.Load().(string)
	return id
}

// Send delivers a text message to a WeChat user via the iLink API.
func (c *WechatChannel) Send(ctx context.Context, msg channel.OutboundMessage) error {
	if !c.IsRunning() || c.api == nil {
		return fmt.Errorf("wechat: channel not running")
	}
	if err := c.ensureSessionActive(); err != nil {
		return err
	}
	if msg.Text == "" {
		return nil
	}

	toUserID := msg.ChatID

	// Retrieve context_token from our per-user map
	contextToken := ""
	if ct, ok := c.contextTokens.Load(toUserID); ok {
		contextToken, _ = ct.(string)
	}
	if contextToken == "" {
		// Also try the state manager
		contextToken = c.stateMgr.GetContextToken(toUserID)
	}
	if contextToken == "" {
		return fmt.Errorf("wechat: missing context token for chat %s", toUserID)
	}

	if err := c.sendTextMessage(ctx, toUserID, contextToken, msg.Text); err != nil {
		logger.ErrorCF("wechat", "failed to send message", map[string]any{
			"to_user_id": toUserID,
			"error":      err.Error(),
		})
		if c.remainingPause() > 0 {
			return fmt.Errorf("wechat: send failed (session paused)")
		}
		return fmt.Errorf("wechat: send failed: %w", err)
	}

	return nil
}

// GetSelfInfo returns the WeChat account ID (as self ID) and nickname.
// Nickname is not available via iLink API, so account_id is used for both.
func (c *WechatChannel) GetSelfInfo() (int64, string) {
	// AccountID may be a string like "wx_xxx", not a numeric int64.
	// Return 0 and the account_id as nickname for display.
	return 0, c.cfg.AccountID
}

// SetConfig updates the channel config at runtime (used by hot-reload).
func (c *WechatChannel) SetConfig(cfg config.WechatChannelConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cfg = cfg
	c.stateMgr = NewWechatStateManager(&cfg)
}

// HasToken returns whether the channel has a valid auth token.
func (c *WechatChannel) HasToken() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cfg.Token != ""
}

// ---------------------------------------------------------------------------
// State persistence
// ---------------------------------------------------------------------------

func (c *WechatChannel) restoreState() error {
	// Restore context tokens
	tokens, err := c.stateMgr.LoadContextTokens()
	if err != nil {
		logger.WarnCF("wechat", "failed to load context tokens", map[string]any{"error": err.Error()})
	} else if len(tokens) > 0 {
		for userID, token := range tokens {
			c.contextTokens.Store(userID, token)
		}
		logger.InfoCF("wechat", "restored context tokens", map[string]any{"count": len(tokens)})
	}

	// Restore sync cursor
	cursor, err := c.stateMgr.LoadCursor()
	if err != nil {
		logger.WarnCF("wechat", "failed to load sync cursor", map[string]any{"error": err.Error()})
	} else if cursor != "" {
		c.syncCursorMu.Lock()
		c.syncCursor = cursor
		c.syncCursorMu.Unlock()
		logger.InfoCF("wechat", "restored sync cursor", map[string]any{"bytes": len(cursor)})
	}

	return nil
}

// ---------------------------------------------------------------------------
// Poll loop
// ---------------------------------------------------------------------------

func (c *WechatChannel) pollLoop(ctx context.Context) {
	const (
		defaultPollTimeoutMs = 35_000
		retryDelay           = 2 * time.Second
		backoffDelay         = 30 * time.Second
		maxConsecutiveFails  = 3
	)

	consecutiveFails := 0

	c.syncCursorMu.Lock()
	getUpdatesBuf := c.syncCursor
	c.syncCursorMu.Unlock()

	nextTimeoutMs := defaultPollTimeoutMs

	for {
		select {
		case <-ctx.Done():
			logger.InfoC("wechat", "poll loop stopped")
			return
		default:
		}

		if err := c.waitWhileSessionPaused(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}

		pollCtx, pollCancel := context.WithTimeout(ctx, time.Duration(nextTimeoutMs+5000)*time.Millisecond)

		resp, err := c.api.GetUpdates(pollCtx, GetUpdatesReq{
			GetUpdatesBuf: getUpdatesBuf,
		})
		pollCancel()

		if err != nil {
			if ctx.Err() != nil {
				return
			}

			consecutiveFails++
			logger.WarnCF("wechat", "getUpdates failed", map[string]any{
				"error":   err.Error(),
				"attempt": consecutiveFails,
			})

			if consecutiveFails >= maxConsecutiveFails {
				logger.ErrorCF("wechat", "too many consecutive failures, backing off", map[string]any{
					"duration": backoffDelay,
				})
				consecutiveFails = 0
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoffDelay):
				}
			} else {
				select {
				case <-ctx.Done():
					return
				case <-time.After(retryDelay):
				}
			}
			continue
		}

		if isSessionExpiredStatus(resp.Ret, resp.Errcode) {
			remaining := c.pauseSession("getupdates", resp.Ret, resp.Errcode, resp.Errmsg)
			select {
			case <-ctx.Done():
				return
			case <-time.After(remaining):
			}
			continue
		}

		if resp.Errcode != 0 || resp.Ret != 0 {
			consecutiveFails++
			logger.ErrorCF("wechat", "getUpdates API error", map[string]any{
				"ret":     resp.Ret,
				"errcode": resp.Errcode,
				"errmsg":  resp.Errmsg,
			})
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryDelay):
			}
			continue
		}

		consecutiveFails = 0
		c.clearPause()

		if resp.LongpollingTimeoutMs > 0 {
			nextTimeoutMs = resp.LongpollingTimeoutMs
		}

		// Advance cursor
		if resp.GetUpdatesBuf != "" {
			getUpdatesBuf = resp.GetUpdatesBuf
			c.syncCursorMu.Lock()
			c.syncCursor = resp.GetUpdatesBuf
			c.syncCursorMu.Unlock()
			if err := c.stateMgr.SaveCursor(resp.GetUpdatesBuf); err != nil {
				logger.WarnCF("wechat", "failed to persist sync cursor", map[string]any{"error": err.Error()})
			}
		}

		// Dispatch messages
		for _, msg := range resp.Msgs {
			c.handleMessage(ctx, msg)
		}
	}
}

// ---------------------------------------------------------------------------
// Message handling
// ---------------------------------------------------------------------------

func (c *WechatChannel) handleMessage(ctx context.Context, msg WeixinMessage) {
	fromUserID := msg.FromUserID
	if fromUserID == "" {
		return
	}

	// Record as notification target for system notifications.
	c.notifyChatID.Store(fromUserID)

	// Build text content from item_list
	var parts []string
	hasMedia := false
	for _, item := range msg.ItemList {
		switch item.Type {
		case MessageItemTypeText:
			if item.TextItem != nil && item.TextItem.Text != "" {
				parts = append(parts, item.TextItem.Text)
			}
		case MessageItemTypeVoice:
			if item.VoiceItem != nil && item.VoiceItem.Text != "" {
				parts = append(parts, item.VoiceItem.Text)
			} else {
				parts = append(parts, "[audio]")
			}
			hasMedia = true
		case MessageItemTypeImage:
			parts = append(parts, "[image]")
			hasMedia = true
		case MessageItemTypeFile:
			if item.FileItem != nil && item.FileItem.FileName != "" {
				parts = append(parts, fmt.Sprintf("[file: %s]", item.FileItem.FileName))
			} else {
				parts = append(parts, "[file]")
			}
			hasMedia = true
		case MessageItemTypeVideo:
			parts = append(parts, "[video]")
			hasMedia = true
		}
	}

	content := strings.Join(parts, "\n")
	if content == "" && !hasMedia {
		return
	}

	senderID := fromUserID
	senderName := fromUserID

	logger.InfoCF("wechat", "message received", map[string]any{
		"sender":  senderID,
		"content": truncate(content, 100),
	})

	// Store context_token for outbound reply association
	if msg.ContextToken != "" {
		c.contextTokens.Store(fromUserID, msg.ContextToken)
		if err := c.stateMgr.SetContextToken(fromUserID, msg.ContextToken); err != nil {
			logger.WarnCF("wechat", "failed to persist context token", map[string]any{"error": err.Error()})
		}
	}

	ctx = c.Context()
	if ctx == nil {
		return
	}

	c.HandleInbound(ctx, channel.InboundMessage{
		Channel:    "wechat",
		ChatID:     fromUserID,
		SenderID:   senderID,
		SenderName: senderName,
		Text:       content,
		MediaType:  "text",
	})
}

// ---------------------------------------------------------------------------
// Sending
// ---------------------------------------------------------------------------

func (c *WechatChannel) sendTextMessage(ctx context.Context, toUserID, contextToken, text string) error {
	req := SendMessageReq{
		Msg: SendMsg{
			ToUserID:     toUserID,
			ClientID:     "homestock-" + uuid.New().String(),
			MessageType:  MessageTypeBot,
			MessageState: MessageStateFinish,
			ItemList: []MessageItem{
				{
					Type: MessageItemTypeText,
					TextItem: &TextItem{
						Text: text,
					},
				},
			},
			ContextToken: contextToken,
		},
	}

	resp, err := c.api.SendMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	if isSessionExpiredStatus(resp.Ret, resp.Errcode) {
		c.pauseSession("sendmessage", resp.Ret, resp.Errcode, resp.Errmsg)
		return fmt.Errorf("session expired (ret=%d errcode=%d)", resp.Ret, resp.Errcode)
	}

	if resp.Ret != 0 || resp.Errcode != 0 {
		return fmt.Errorf("send message API error: ret=%d errcode=%d errmsg=%s", resp.Ret, resp.Errcode, resp.Errmsg)
	}

	c.clearPause()
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
