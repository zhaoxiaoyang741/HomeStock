package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdispatcher "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/channel"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

const errCodeTenantTokenInvalid = 99991663

type FeishuChannel struct {
	*channel.BaseChannel
	appID      string
	appSecret  string
	client     *lark.Client
	wsClient   *larkws.Client
	tokenCache *tokenCache

	botOpenID    atomic.Value
	notifyChatID atomic.Value

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewFeishuChannel(appID, appSecret string) *FeishuChannel {
	base := &channel.BaseChannel{}
	base.InitBase("feishu", nil)

	tc := newTokenCache()
	return &FeishuChannel{
		BaseChannel: base,
		appID:       appID,
		appSecret:   appSecret,
		tokenCache:  tc,
		client: lark.NewClient(
			appID,
			appSecret,
			lark.WithTokenCache(tc),
		),
	}
}

func (c *FeishuChannel) Start(ctx context.Context) error {
	if c.appID == "" || c.appSecret == "" {
		return fmt.Errorf("feishu: app_id or app_secret is empty")
	}

	c.BaseStart(ctx)

	if err := c.fetchBotOpenID(ctx); err != nil {
		logger.WarnCF("feishu", "failed to fetch bot open_id, @mention detection may not work", map[string]any{
			"error": err.Error(),
		})
	}

	dispatcher := larkdispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(c.handleMessageReceive)

	runCtx, cancel := context.WithCancel(ctx)

	c.mu.Lock()
	c.cancel = cancel
	c.wsClient = larkws.NewClient(
		c.appID,
		c.appSecret,
		larkws.WithEventHandler(dispatcher),
	)
	wsClient := c.wsClient
	c.mu.Unlock()

	logger.InfoC("feishu", "Feishu channel started (websocket mode)")

	go func() {
		for {
			err := wsClient.Start(runCtx)
			c.mu.Lock()
			isRunning := c.cancel != nil
			c.mu.Unlock()
			if !isRunning {
				return // channel was stopped
			}
			if err != nil {
				logger.WarnCF("feishu", "WebSocket disconnected, reconnecting in 5s...", map[string]any{
					"error": err.Error(),
				})
			}
			select {
			case <-runCtx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	}()

	return nil
}

func (c *FeishuChannel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.wsClient = nil
	c.mu.Unlock()

	c.BaseStop()
	logger.InfoC("feishu", "Feishu channel stopped")
	return nil
}

func (c *FeishuChannel) Send(ctx context.Context, msg channel.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("feishu: channel not running")
	}
	if msg.ChatID == "" {
		return fmt.Errorf("feishu: chat ID is empty")
	}

	cardContent, err := buildMarkdownCard(msg.Text)
	if err != nil {
		return c.sendText(ctx, msg.ChatID, msg.Text)
	}

	err = c.sendCard(ctx, msg.ChatID, cardContent)
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "11310") {
		logger.WarnCF("feishu", "card send failed (table limit), falling back to text", map[string]any{
			"chat_id": msg.ChatID,
			"error":   errMsg,
		})
		return c.sendText(ctx, msg.ChatID, msg.Text)
	}

	return err
}

func (c *FeishuChannel) handleMessageReceive(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}

	// The SDK's ctx is tied to this wsClient instance. When the channel is
	// stopped and restarted, the old wsClient still holds a canceled context
	// but its WebSocket connection remains open (the SDK's Start() blocks
	// forever with select{} and provides no public Close/Stop method). Any
	// messages delivered to the stale connection would fail with "context
	// canceled". Use the channel's current context instead so that inbound
	// messages always flow through a valid context, regardless of which
	// wsClient delivered them.
	//
	// TODO: remove this workaround once the lark SDK exposes a public
	// disconnect API (or Start respects ctx.Done()).
	channelCtx := c.Context()
	if channelCtx == nil {
		return nil
	}
	select {
	case <-channelCtx.Done():
		return nil
	default:
	}

	message := event.Event.Message
	sender := event.Event.Sender

	chatID := stringValue(message.ChatId)
	if chatID == "" {
		return nil
	}

	// Record as notification target for system notifications.
	c.notifyChatID.Store(chatID)

	senderID := extractFeishuSenderID(sender)
	if senderID == "" {
		senderID = "unknown"
	}

	messageType := stringValue(message.MessageType)
	rawContent := stringValue(message.Content)

	senderName := senderID
	if sender != nil && sender.SenderId != nil {
		if sender.SenderId.UserId != nil && *sender.SenderId.UserId != "" {
			senderName = *sender.SenderId.UserId
		} else if sender.SenderId.OpenId != nil && *sender.SenderId.OpenId != "" {
			senderName = *sender.SenderId.OpenId
		}
	}

	content := extractContent(messageType, rawContent)

	chatType := stringValue(message.ChatType)
	isGroup := chatType == "group" || chatType == "chat"

	if isGroup {
		isMentioned := c.isBotMentioned(message)
		if !isMentioned {
			// Phase 1a: keyword-based wake-up — process inventory-related
			// messages even without @mention to reduce input friction.
			if !hasInventoryKeyword(content) {
				return nil
			}
		}
		if len(message.Mentions) > 0 {
			content = stripMentionPlaceholders(content, message.Mentions)
		}
	}

	mediaType := "text"
	fileKey := ""
	switch messageType {
	case larkim.MsgTypeImage:
		mediaType = "image"
		fileKey = extractImageKey(rawContent)
		content = ""
	case larkim.MsgTypeAudio:
		mediaType = "voice"
		fileKey = extractFileKey(rawContent)
		content = ""
	case larkim.MsgTypeMedia:
		mediaType = "video"
		fileKey = extractFileKey(rawContent)
		content = ""
	case larkim.MsgTypeFile:
		mediaType = "file"
		fileKey = extractFileKey(rawContent)
	}

	if content == "" && mediaType == "text" {
		content = "[empty message]"
	}

	fmt.Printf("[%s] 📩 %s (%s): %s\n",
		time.Now().Format("15:04:05"),
		senderName, chatID,
		content)

	logger.InfoCF("feishu", "message received", map[string]any{
		"sender_id":    senderID,
		"sender_name":  senderName,
		"chat_id":      chatID,
		"chat_type":    chatType,
		"message_type": messageType,
		"content_len":  len(content),
	})

	inbound := channel.InboundMessage{
		Channel:    "feishu",
		ChatID:     chatID,
		SenderID:   senderID,
		SenderName: senderName,
		Text:       content,
		MediaType:  mediaType,
		FileKey:    fileKey,
	}

	c.HandleInbound(channelCtx, inbound)
	return nil
}

func (c *FeishuChannel) fetchBotOpenID(ctx context.Context) error {
	resp, err := c.client.Do(ctx, &larkcore.ApiReq{
		HttpMethod:                http.MethodGet,
		ApiPath:                   "/open-apis/bot/v3/info",
		SupportedAccessTokenTypes: []larkcore.AccessTokenType{larkcore.AccessTokenTypeTenant},
	})
	if err != nil {
		return fmt.Errorf("bot info request: %w", err)
	}

	var result struct {
		Code int `json:"code"`
		Bot  struct {
			OpenID string `json:"open_id"`
		} `json:"bot"`
	}
	if err := json.Unmarshal(resp.RawBody, &result); err != nil {
		return fmt.Errorf("bot info parse: %w", err)
	}
	if result.Code != 0 {
		c.invalidateTokenOnAuthError(result.Code)
		return fmt.Errorf("bot info api error (code=%d)", result.Code)
	}
	if result.Bot.OpenID == "" {
		return fmt.Errorf("bot info: empty open_id")
	}

	c.botOpenID.Store(result.Bot.OpenID)
	logger.InfoCF("feishu", "fetched bot open_id", map[string]any{
		"open_id": result.Bot.OpenID,
	})
	return nil
}

func (c *FeishuChannel) isBotMentioned(message *larkim.EventMessage) bool {
	if message.Mentions == nil {
		return false
	}
	knownID, _ := c.botOpenID.Load().(string)
	if knownID == "" {
		logger.DebugC("feishu", "bot open_id unknown, cannot detect @mention")
		return false
	}
	for _, m := range message.Mentions {
		if m.Id == nil {
			continue
		}
		if m.Id.OpenId != nil && *m.Id.OpenId == knownID {
			return true
		}
	}
	return false
}

func (c *FeishuChannel) sendCard(ctx context.Context, chatID, cardContent string) error {
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeInteractive).
			Content(cardContent).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu send card: %w", err)
	}
	if !resp.Success() {
		c.invalidateTokenOnAuthError(resp.Code)
		return fmt.Errorf("feishu card api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}

	logger.DebugCF("feishu", "card message sent", map[string]any{"chat_id": chatID})
	return nil
}

func (c *FeishuChannel) sendText(ctx context.Context, chatID, text string) error {
	content, _ := json.Marshal(map[string]string{"text": text})

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeText).
			Content(string(content)).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu send text: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("feishu text api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}

	logger.DebugCF("feishu", "text message sent (fallback)", map[string]any{"chat_id": chatID})
	return nil
}

func (c *FeishuChannel) invalidateTokenOnAuthError(code int) {
	if code == errCodeTenantTokenInvalid {
		c.tokenCache.InvalidateAll()
		logger.WarnC("feishu", "invalidated cached token due to auth error")
	}
}

// Reconfigure updates the channel credentials at runtime and restarts if enabled.
// The inbound handler set via SetInboundHandler is preserved.
func (c *FeishuChannel) Reconfigure(ctx context.Context, appID, appSecret string, enabled bool) error {
	if c.IsRunning() {
		if err := c.Stop(ctx); err != nil {
			return fmt.Errorf("reconfigure: stop: %w", err)
		}
	}

	c.mu.Lock()
	c.appID = appID
	c.appSecret = appSecret
	c.client = lark.NewClient(appID, appSecret, lark.WithTokenCache(c.tokenCache))
	c.tokenCache.InvalidateAll()
	c.botOpenID.Store("")
	c.mu.Unlock()

	if enabled {
		if err := c.Start(ctx); err != nil {
			return fmt.Errorf("reconfigure: start: %w", err)
		}
	}

	return nil
}

// NotifyChatID returns the ChatID of the last conversation that received a message.
// Implements channel.NotifyTargetProvider for system notifications.
func (c *FeishuChannel) NotifyChatID() string {
	id, _ := c.notifyChatID.Load().(string)
	return id
}

// GetTokenCache returns the internal token cache for seeding OAuth tokens.
func (c *FeishuChannel) GetTokenCache() *tokenCache { return c.tokenCache }

var _ channel.Channel = (*FeishuChannel)(nil)
