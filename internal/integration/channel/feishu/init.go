package feishu

import (
	"encoding/json"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/channel"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func init() {
	channel.RegisterFactory("feishu", func(raw json.RawMessage) (channel.Channel, error) {
		var cfg config.FeishuChannelConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, err
		}
		if !cfg.Enabled {
			return nil, nil
		}
		return NewFeishuChannel(cfg.AppID, cfg.AppSecret), nil
	})
}
