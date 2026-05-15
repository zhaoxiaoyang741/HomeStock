package wechat

import (
	"context"
	"fmt"
	"sync"

	"github.com/eatmoreapple/openwechat"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/channel"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

const (
	storageFilename = "data/wechat_storage.json"
)

// WechatChannel implements channel.Channel for personal WeChat via openwechat.
type WechatChannel struct {
	*channel.BaseChannel
	bot       *openwechat.Bot
	self      *openwechat.Self
	loginSess *LoginSession
	stopped   chan struct{}

	mu         sync.Mutex
	startOnce  sync.Once
	stopOnce   sync.Once
}

// NewWechatChannel creates a new WechatChannel.
func NewWechatChannel() *WechatChannel {
	c := &WechatChannel{
		BaseChannel: &channel.BaseChannel{},
		loginSess:   &LoginSession{},
		stopped:     make(chan struct{}),
	}
	c.InitBase("wechat", nil)
	return c
}

// Name returns the channel name.
func (c *WechatChannel) Name() string { return "wechat" }

// Start establishes the WeChat connection via openwechat.
func (c *WechatChannel) Start(ctx context.Context) error {
	c.BaseStart(ctx)
	c.mu.Lock()
	c.bot = openwechat.NewBot(ctx)
	bot := c.bot
	c.mu.Unlock()

	// Set up QR code UUID callback
	bot.UUIDCallback = func(uuid string) {
		c.mu.Lock()
		c.loginSess = NewLoginSession(uuid)
		c.mu.Unlock()
		logger.InfoCF("wechat", "QR code ready, waiting for scan", map[string]any{
			"uuid": uuid,
		})
	}

	// Scan callback
	bot.ScanCallBack = func(_ openwechat.CheckLoginResponse) {
		c.mu.Lock()
		c.loginSess.SetStatus(LoginStatusScanned)
		c.mu.Unlock()
		logger.InfoC("wechat", "QR code scanned, please confirm on phone")
	}

	// Login callback
	bot.LoginCallBack = func(_ openwechat.CheckLoginResponse) {
		c.mu.Lock()
		c.loginSess.SetStatus(LoginStatusSuccess)
		c.mu.Unlock()
		logger.InfoC("wechat", "Login successful")
	}

	// Logout callback
	bot.LogoutCallBack = func(b *openwechat.Bot) {
		logger.InfoC("wechat", "Bot logged out")
	}

	// Message handler
	bot.MessageHandler = func(msg *openwechat.Message) {
		c.handleMessage(msg)
	}

	// Try hot login first (reuse saved session), fall back to QR login
	storage := openwechat.NewFileHotReloadStorage(storageFilename)
	if err := bot.HotLogin(storage); err != nil {
		logger.InfoC("wechat", "Hot login failed, trying QR login")
		// Reset login session for QR login
		c.mu.Lock()
		c.loginSess = &LoginSession{}
		c.mu.Unlock()

		if err := bot.Login(); err != nil {
			close(c.stopped)
			return fmt.Errorf("wechat: qr login failed: %w", err)
		}
	}

	// Get current user info
	self, err := bot.GetCurrentUser()
	if err != nil {
		close(c.stopped)
		return fmt.Errorf("wechat: get current user: %w", err)
	}
	c.mu.Lock()
	c.self = self
	c.mu.Unlock()

	logger.InfoCF("wechat", "Bot logged in as %s", map[string]any{
		"nickname": self.NickName,
	})

	// Block until bot exits (message sync runs in background goroutine)
	go func() {
		if err := bot.Block(); err != nil {
			logger.InfoCF("wechat", "Bot exited", map[string]any{
				"error": err.Error(),
			})
		} else {
			logger.InfoC("wechat", "Bot exited normally")
		}
		close(c.stopped)
	}()

	return nil
}

// Stop gracefully shuts down the WeChat connection.
func (c *WechatChannel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.bot != nil {
		c.bot.Exit()
	}
	c.mu.Unlock()

	// Wait for bot to fully stop
	select {
	case <-c.stopped:
	case <-ctx.Done():
	}

	c.BaseStop()
	logger.InfoC("wechat", "Channel stopped")
	return nil
}

// Send delivers an outbound message through WeChat.
func (c *WechatChannel) Send(ctx context.Context, msg channel.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("wechat: channel not running")
	}
	if msg.ChatID == "" {
		return fmt.Errorf("wechat: chat ID is empty")
	}
	if msg.Text == "" {
		return nil
	}

	c.mu.Lock()
	self := c.self
	bot := c.bot
	c.mu.Unlock()

	if self == nil || bot == nil {
		return fmt.Errorf("wechat: not logged in")
	}

	// Send text directly via bot caller — only needs UserName, no Friend/Group object
	sendMsg := openwechat.NewTextSendMessage(msg.Text, self.UserName, msg.ChatID)
	_, err := bot.Caller.WebWxSendMsg(ctx, &openwechat.CallerWebWxSendMsgOptions{
		LoginInfo:   bot.Storage.LoginInfo,
		BaseRequest: bot.Storage.Request,
		Message:     sendMsg,
	})
	if err != nil {
		return fmt.Errorf("wechat: send message: %w", err)
	}
	return nil
}

// GetLoginSession returns a snapshot of the current login session state.
func (c *WechatChannel) GetLoginSession() LoginSession {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.loginSess == nil {
		return LoginSession{}
	}
	return c.loginSess.Snapshot()
}

// IsLoggedIn returns true if the bot has successfully logged in.
func (c *WechatChannel) IsLoggedIn() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.self != nil
}

// handleMessage processes an incoming WeChat message and pushes it to the bus.
func (c *WechatChannel) handleMessage(msg *openwechat.Message) {
	// Only process text messages from friends or self
	if !msg.IsText() {
		return
	}

	// Skip messages sent by self (echo)
	if msg.IsSendBySelf() {
		return
	}

	// Skip system messages
	if msg.IsSystem() {
		return
	}

	ctx := c.Context()
	if ctx == nil {
		return
	}

	chatID := msg.FromUserName
	if msg.IsSendByGroup() {
		// Get group UserName for reply routing
		receiver, err := msg.Receiver()
		if err == nil && receiver != nil {
			chatID = receiver.UserName
		}
	}

	senderName := msg.FromUserName
	sender, err := msg.Sender()
	if err == nil && sender != nil {
		senderName = sender.NickName
	}

	content := msg.Content

	logger.InfoCF("wechat", "message received", map[string]any{
		"sender":  senderName,
		"chat_id": chatID,
		"content": content,
	})

	// Push into the message bus via BaseChannel's inbound handler
	c.HandleInbound(ctx, channel.InboundMessage{
		Channel:    "wechat",
		ChatID:     chatID,
		SenderID:   msg.FromUserName,
		SenderName: senderName,
		Text:       content,
		MediaType:  "text",
	})
}
