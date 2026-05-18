package wechat

import (
	"encoding/json"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/channel"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func init() {
	channel.RegisterFactory("wechat", func(raw json.RawMessage) (channel.Channel, error) {
		var cfg config.WechatChannelConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, err
		}
		if !cfg.Enabled {
			return nil, nil
		}
		return NewWechatChannel(cfg), nil
	})
}
