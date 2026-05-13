package feishu

import "context"

// MediaProcessor handles media content from Feishu messages.
//
// Reserved for future ASR (speech-to-text) and OCR integration.
// Currently all media messages are replied with "暂不支持" (not supported).
type MediaProcessor interface {
	ProcessVoice(ctx context.Context, fileKey string) (string, error)
	ProcessImage(ctx context.Context, fileKey string) (string, error)
}

// defaultMediaProcessor returns a stub processor that always returns an error.
// Replace with real ASR/OCR implementations in future iterations.
func defaultMediaProcessor() MediaProcessor {
	return &stubMediaProcessor{}
}

type stubMediaProcessor struct{}

func (s *stubMediaProcessor) ProcessVoice(_ context.Context, fileKey string) (string, error) {
	return "", nil
}

func (s *stubMediaProcessor) ProcessImage(_ context.Context, fileKey string) (string, error) {
	return "", nil
}
