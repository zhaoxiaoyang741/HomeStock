package app

import (
	"context"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/channel"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/bus"
	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/channel/feishu"
	"github.com/zhaoxiaoyang741/HomeStock/internal/handler"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

func initChannels(
	cfg config.ChannelsConfig,
	msgBus *bus.MessageBus,
	uow *gormrepo.UnitOfWork,
	configPath string,
) (
	channelMgr *channel.Manager,
	feishuH *handler.FeishuHandler,
	feishuCh *feishu.FeishuChannel,
	oauthSvc *feishu.OAuthService,
) {
	channelMgr = channel.NewManager()

	// Inbound handler: routes channel messages into the agent MessageBus
	inboundHandler := func(ctx context.Context, msg channel.InboundMessage) {
		if err := msgBus.PublishInbound(ctx, bus.InboundMessage{
			Channel:    msg.Channel,
			ChatID:     msg.ChatID,
			SenderID:   msg.SenderID,
			SenderName: msg.SenderName,
			Text:       msg.Text,
			MediaType:  msg.MediaType,
			FileKey:    msg.FileKey,
		}); err != nil {
			logger.ErrorCF("app", "publish inbound failed", map[string]any{
				"channel": msg.Channel,
				"error":   err.Error(),
			})
		}
	}

	fc := feishu.NewFeishuChannel(cfg.Feishu.AppID, cfg.Feishu.AppSecret)
	fc.SetInboundHandler(inboundHandler)
	if cfg.Feishu.Enabled {
		channelMgr.AddChannel(fc)
		logger.InfoCF("app", "Feishu channel enabled", nil)
	}

	oauthSvc = feishu.NewOAuthService(
		cfg.Feishu.AppID,
		cfg.Feishu.AppSecret,
		cfg.Feishu.RedirectURI,
		cfg.Feishu.FrontendURL,
		uow.Repos().SystemSettings(),
	)

	feishuCh = fc
	feishuH = handler.NewFeishuHandler(oauthSvc, channelMgr, fc, cfg.Feishu.FrontendURL, configPath)

	// Channel update callback: reconfigures Feishu channel when settings change
	feishuH.SetChannelUpdateFn(func(feishuCfg config.FeishuChannelConfig) error {
		ctx := context.Background()

		if err := fc.Reconfigure(ctx, feishuCfg.AppID, feishuCfg.AppSecret, feishuCfg.Enabled); err != nil {
			return err
		}

		oauthSvc.UpdateCredentials(feishuCfg.AppID, feishuCfg.AppSecret)

		if err := oauthSvc.ClearAuth(ctx); err != nil {
			logger.WarnCF("feishu", "failed to clear stale oauth token", map[string]any{"error": err.Error()})
		}

		if feishuCfg.Enabled {
			if _, exists := channelMgr.GetChannel("feishu"); !exists {
				channelMgr.AddChannel(fc)
			}
		} else {
			channelMgr.RemoveChannel("feishu")
		}

		return nil
	})

	return
}
