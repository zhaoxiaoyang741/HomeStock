# Phase 1a 语音识别实施设计

## 架构决策

**路径：** 不走飞书 ASR → 不依赖飞书的语音转文字 API（避免 PCM 转码复杂度）
**方案：** 在 Feishu channel 下载语音文件 → 直接发给 Whisper 兼容的语音识别模型 → 获取转写文本
**配置：** 在 `model_list` 中通过 `model_name: "speech"` 维护语音模型，和 chat 模型并列

## 整体流程

```
Feishu WebSocket 收到语音消息
         │
         ▼
handleMessageReceive
  ├─ 提取 message_id (新增) + file_key (已有)
  │
  ├─ [voice 分支] MediaProcessor.ProcessVoice(messageID, fileKey)
  │   ├─ Step 1: MessageResource.Get(message_id, file_key, type="file")
  │   │          下载语音文件 (AMR/OPUS/MP3 等原生格式)
  │   ├─ Step 2: WhisperProvider.SpeechToText(audio bytes)
  │   │          POST /v1/audio/transcriptions (multipart/form-data)
  │   │          设 language=zh, response_format=text
  │   └─ 成功 → text, nil
  │
  ├─ 成功 → content = 转写文本, mediaType = "text"
  │          → InboundMessage{Text: 转写结果, MediaType: "text"}
  │          → bus → AgentLoop → 正常处理 (AgentLoop 无需感知语音)
  │
  └─ 失败 → 直接 c.sendText() 回复错误提示
            → return nil (不 publish 到 bus)
```

## 改动清单

### 1. `pkg/llm/provider.go` — 新增 SpeechProvider 接口

```go
// SpeechProvider handles speech-to-text transcription.
// Separate from LLMProvider because the API shape is different
// (multipart form-data vs JSON chat completions).
type SpeechProvider interface {
    // SpeechToText transcribes audio bytes and returns the recognized text.
    // audio is the raw audio file content in any common format
    // (AMR, OPUS, MP3, WAV, etc — Whisper supports all major formats).
    SpeechToText(ctx context.Context, audio []byte, model string) (string, error)
}
```

不往 `LLMProvider` 上加方法。Whisper 是 `multipart/form-data` 请求，和 Chat 的 JSON 请求完全不同，拆开更干净。只有 OpenAI 兼容的 Provider 才需要实现这个接口。

### 2. `pkg/llm/whisper.go` — 新建文件

WhisperProvider 实现，调用 OpenAI 兼容的 `POST /v1/audio/transcriptions`：

```go
package llm

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "strings"
    "time"

    "github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

const defaultWhisperTimeout = 120 * time.Second

type WhisperProvider struct {
    apiKey  string
    apiBase string
    client  *http.Client
}

func NewWhisperProvider(cfg config.ModelConfig) *WhisperProvider {
    base := strings.TrimRight(cfg.APIBase, "/")
    if base == "" {
        base = "https://api.openai.com/v1"
    }
    return &WhisperProvider{
        apiKey:  cfg.APIKey,
        apiBase: base,
        client:  &http.Client{Timeout: defaultWhisperTimeout},
    }
}

func (p *WhisperProvider) SpeechToText(ctx context.Context, audio []byte, model string) (string, error) {
    if model == "" {
        model = "whisper-1"
    }

    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    // model field
    if err := writer.WriteField("model", model); err != nil {
        return "", fmt.Errorf("whisper: write model field: %w", err)
    }
    // language hint — Chinese is the primary use case
    if err := writer.WriteField("language", "zh"); err != nil {
        return "", fmt.Errorf("whisper: write language field: %w", err)
    }
    // response format — plain text, not JSON
    if err := writer.WriteField("response_format", "text"); err != nil {
        return "", fmt.Errorf("whisper: write format field: %w", err)
    }
    // file field
    part, err := writer.CreateFormFile("file", "audio")
    if err != nil {
        return "", fmt.Errorf("whisper: create form file: %w", err)
    }
    if _, err := part.Write(audio); err != nil {
        return "", fmt.Errorf("whisper: write audio: %w", err)
    }
    if err := writer.Close(); err != nil {
        return "", fmt.Errorf("whisper: close writer: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost,
        p.apiBase+"/audio/transcriptions", body)
    if err != nil {
        return "", fmt.Errorf("whisper: create request: %w", err)
    }
    req.Header.Set("Content-Type", writer.FormDataContentType())
    if p.apiKey != "" {
        req.Header.Set("Authorization", "Bearer "+p.apiKey)
    }

    resp, err := p.client.Do(req)
    if err != nil {
        return "", fmt.Errorf("whisper: request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        raw, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("whisper: API error (status=%d): %s",
            resp.StatusCode, truncate(string(raw), 200))
    }

    raw, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("whisper: read response: %w", err)
    }

    text := strings.TrimSpace(string(raw))
    if text == "" {
        return "", fmt.Errorf("whisper: empty transcription")
    }
    return text, nil
}
```

### 3. `pkg/llm/provider_factory.go` — 新增工厂函数

```go
// NewSpeechProvider creates a SpeechProvider from a model config.
// Supported providers: "openai" (Whisper-compatible, default).
func NewSpeechProvider(cfg config.ModelConfig) (SpeechProvider, error) {
    switch cfg.Provider {
    case "", "openai":
        return NewWhisperProvider(cfg), nil
    default:
        return nil, fmt.Errorf("llm: unsupported speech provider %q", cfg.Provider)
    }
}
```

### 4. `pkg/config/config.go` — 新增查找辅助函数

```go
// FindModelByName returns the first model config with the given model_name, or nil.
func FindModelByName(models []ModelConfig, name string) *ModelConfig {
    for i := range models {
        if models[i].ModelName == name {
            return &models[i]
        }
    }
    return nil
}
```

Config 示例：

```json
{
  "model_list": [
    {
      "model_name": "chat",
      "model": "deepseek-chat",
      "provider": "deepseek",
      "api_key": "sk-xxx",
      "enabled": true
    },
    {
      "model_name": "speech",
      "model": "whisper-1",
      "provider": "openai",
      "api_key": "sk-xxx",
      "enabled": true
    }
  ]
}
```

### 5. `internal/integration/channel/feishu/media.go` — 重构

从 interface + stub 改为具体 struct：

```go
package feishu

import (
    "context"
    "fmt"
    "io"

    lark "github.com/larksuite/oapi-sdk-go/v3"
    larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"

    "github.com/zhaoxiaoyang741/HomeStock/pkg/llm"
    "github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// MediaProcessor handles media content from Feishu messages.
type MediaProcessor struct {
    client         *lark.Client
    speechProvider llm.SpeechProvider
    speechModel    string
}

// NewMediaProcessor creates a MediaProcessor with the given dependencies.
func NewMediaProcessor(client *lark.Client, speechProvider llm.SpeechProvider, speechModel string) *MediaProcessor {
    return &MediaProcessor{
        client:         client,
        speechProvider: speechProvider,
        speechModel:    speechModel,
    }
}

// ProcessVoice downloads voice message and transcribes via speech model.
func (p *MediaProcessor) ProcessVoice(ctx context.Context, messageID, fileKey string) (string, error) {
    req := larkim.NewGetMessageResourceReqBuilder().
        MessageId(messageID).
        FileKey(fileKey).
        Type("file"). // voice is type=file
        Build()

    resp, err := p.client.Im.V1.MessageResource.Get(ctx, req)
    if err != nil {
        return "", fmt.Errorf("download voice: %w", err)
    }
    if resp.File == nil {
        return "", fmt.Errorf("download voice: empty response")
    }

    audioBytes, err := io.ReadAll(resp.File)
    if err != nil {
        return "", fmt.Errorf("read voice: %w", err)
    }

    logger.InfoCF("feishu", "voice file downloaded", map[string]any{
        "message_id": messageID,
        "size":       len(audioBytes),
    })

    text, err := p.speechProvider.SpeechToText(ctx, audioBytes, p.speechModel)
    if err != nil {
        return "", fmt.Errorf("speech to text: %w", err)
    }

    return text, nil
}
```

删除 `defaultMediaProcessor()`、`stubMediaProcessor` 和 `MediaProcessor` interface（不再需要，由具体 struct 替代）。

### 6. `internal/integration/channel/feishu/channel.go` — 修改

**a) FeishuChannel 新增字段：**

```go
type FeishuChannel struct {
    // ... existing fields ...
    mediaProcessor *MediaProcessor
}
```

**b) Setter 方法：**

```go
func (c *FeishuChannel) SetMediaProcessor(mp *MediaProcessor) {
    c.mediaProcessor = mp
}
```

**c) Client() 访问器（用于 server.go 中组装 MediaProcessor）：**

```go
func (c *FeishuChannel) Client() *lark.Client {
    return c.client
}
```

**d) handleMessageReceive 提取 message_id：**

```go
messageID := stringValue(message.MessageId)
```

**e) Voice 分支修改（现有 `case larkim.MsgTypeAudio:`）：**

```go
case larkim.MsgTypeAudio:
    mediaType = "voice"
    fileKey = extractFileKey(rawContent)
    content = ""

    // Voice → text via speech model (Phase 1a)
    if c.mediaProcessor != nil && messageID != "" && fileKey != "" {
        transcribedText, err := c.mediaProcessor.ProcessVoice(channelCtx, messageID, fileKey)
        if err != nil {
            logger.WarnCF("feishu", "voice transcription failed", map[string]any{
                "chat_id":    chatID,
                "message_id": messageID,
                "error":      err.Error(),
            })
            _ = c.sendText(channelCtx, chatID, "语音识别失败，请重试或输入文字。")
            return nil
        }
        if transcribedText == "" {
            _ = c.sendText(channelCtx, chatID, "没有识别到语音内容，请重试。")
            return nil
        }
        content = transcribedText
        mediaType = "text" // 转成文本，AgentLoop 无需感知语音
        logger.InfoCF("feishu", "voice transcribed", map[string]any{
            "chat_id": chatID,
            "text":    transcribedText,
        })
    }
```

### 7. `cmd/server/internal/server.go` — Composition Root 修改

**a) 在 initAgent 之后、initChannels 之前初始化 speech provider：**

```go
// Speech recognition provider (optional)
var (
    speechProvider llm.SpeechProvider
    speechModel    string
)
if speechCfg := config.FindModelByName(cfg.ModelList, "speech"); speechCfg != nil && speechCfg.Enabled {
    sp, err := llm.NewSpeechProvider(*speechCfg)
    if err != nil {
        return nil, fmt.Errorf("app: create speech provider: %w", err)
    }
    speechProvider = sp
    speechModel = speechCfg.Model
    logger.InfoCF("app", "speech recognition enabled", map[string]any{
        "model": speechModel,
    })
}
```

**b) initChannels 签名增加参数：**

```go
func initChannels(
    cfg config.ChannelsConfig,
    msgBus *bus.MessageBus,
    uow *gormrepo.UnitOfWork,
    configPath string,
    speechProvider llm.SpeechProvider,
    speechModel string,
) ( ... )
```

**c) 创建 FeishuChannel 后设置 MediaProcessor：**

```go
fc := feishu.NewFeishuChannel(cfg.Feishu.AppID, cfg.Feishu.AppSecret)
fc.SetInboundHandler(inboundHandler)

// Wire speech recognition (Phase 1a)
if speechProvider != nil {
    fc.SetMediaProcessor(feishu.NewMediaProcessor(fc.Client(), speechProvider, speechModel))
}
```

## 边界情况处理

| 场景 | 行为 |
|------|------|
| 语音文件下载失败 | `sendText("语音消息处理失败，请重试")`，不 publish 到 bus |
| Whisper API 超时 (120s timeout) | `sendText("语音识别服务暂时不可用")` |
| 空音频/静音 | Whisper 返回空字符串 → `sendText("没有识别到语音内容")` |
| 未配置 speech model | `mediaProcessor` 为 nil → 走现有逻辑 |
| 配置了 speech model 但 API key 错误 | 下载成功但 ASR 返回 401/403 → `sendText("语音识别失败")` |
| 长音频 (>25MB) | Whisper 限制 25MB；MessageResource.Get 也会在下载时超时，回复通用错误 |

## 与现有代码的交互

- **AgentLoop 完全不需要改** — 语音转文字在 channel 层完成，AgentLoop 收到的已是文本（mediaType="text"）
- **bus.InboundMessage 不需要改** — 仍是 `Text` + `MediaType` 结构
- **Hot-reload 不受影响** — speech model 变更需要重启，与 chat model 行为一致
- **`[voice消息暂不支持]` 自动退役** — 配置了 speech model 后，语音消息不会走到那段代码

## 未覆盖（后续阶段）

- **图片识别** — Phase 1b 处理，`MediaProcessor` struct 保留扩展 `ProcessImage()`
- **批量输入确认** — 属于 Phase 1a 的 two-phase confirmation 部分，与语音识别独立
- **Telegram 语音** — Phase 2 处理，届时可为 Telegram 实现其平台的原生 ASR
