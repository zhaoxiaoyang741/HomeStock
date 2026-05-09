package feishu

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

const (
	oauthStateKey = "feishu_oauth_state"
	oauthTokenKey = "feishu_oauth"
)

// StoredOAuthToken holds the OAuth token persisted to SystemSetting.
type StoredOAuthToken struct {
	TenantAccessToken string `json:"tenant_access_token,omitempty"`
	ExpiresAt         int64  `json:"expires_at,omitempty"`
	AppID             string `json:"app_id"`
	AuthorizedAt      int64  `json:"authorized_at,omitempty"`
	BotName           string `json:"bot_name,omitempty"`
}

// FeishuStatus is the response for the status endpoint.
type FeishuStatus struct {
	Configured bool   `json:"configured"`
	Connected  bool   `json:"connected"`
	BotName    string `json:"bot_name,omitempty"`
	AppID      string `json:"app_id"`
}

// OAuthService handles Feishu OAuth authorization code flow and token persistence.
type OAuthService struct {
	appID       string
	appSecret   string
	redirectURI string
	frontendURL string
	repo        repository.SystemSettingRepo
	httpClient  *http.Client

	// cached bot name, populated after successful OAuth
	botName string
}

// NewOAuthService creates a new OAuthService.
func NewOAuthService(appID, appSecret, redirectURI, frontendURL string, repo repository.SystemSettingRepo) *OAuthService {
	return &OAuthService{
		appID:       appID,
		appSecret:   appSecret,
		redirectURI: redirectURI,
		frontendURL: frontendURL,
		repo:        repo,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// GetAuthURL generates a state nonce, persists it, and returns the Feishu OAuth URL.
func (s *OAuthService) GetAuthURL(ctx context.Context) (string, error) {
	state, err := generateState()
	if err != nil {
		return "", fmt.Errorf("feishu oauth: generate state: %w", err)
	}

	if err := s.repo.Upsert(&model.SystemSetting{
		Key:     oauthStateKey,
		Payload: state,
	}); err != nil {
		return "", fmt.Errorf("feishu oauth: store state: %w", err)
	}

	params := url.Values{}
	params.Set("redirect_uri", s.redirectURI)
	params.Set("app_id", s.appID)
	params.Set("state", state)

	return fmt.Sprintf("https://open.feishu.cn/open-apis/authen/v1/index?%s", params.Encode()), nil
}

// HandleCallback validates the state, exchanges the authorization code for a token,
// and persists the token to SystemSetting.
func (s *OAuthService) HandleCallback(ctx context.Context, code, state string) error {
	// Validate state
	stored, err := s.repo.GetByKey(oauthStateKey)
	if err != nil {
		return fmt.Errorf("feishu oauth: get stored state: %w", err)
	}
	if stored.Payload == "" || stored.Payload != state {
		return fmt.Errorf("feishu oauth: state mismatch or expired")
	}

	// Exchange code for tenant_access_token
	token, err := s.exchangeCode(ctx, code)
	if err != nil {
		return fmt.Errorf("feishu oauth: exchange code: %w", err)
	}

	// Persist token
	raw, _ := json.Marshal(token)
	if err := s.repo.Upsert(&model.SystemSetting{
		Key:     oauthTokenKey,
		Payload: string(raw),
	}); err != nil {
		return fmt.Errorf("feishu oauth: store token: %w", err)
	}

	// Cache bot name in memory so GetStatus doesn't need a DB query
	s.botName = token.BotName

	logger.InfoC("feishu", "OAuth token stored successfully")
	return nil
}

// GetStatus returns the current Feishu channel status without querying the DB.
// The bot name is served from the in-memory cache set during HandleCallback.
func (s *OAuthService) GetStatus(_ context.Context, isConnected bool) (*FeishuStatus, error) {
	status := &FeishuStatus{
		Configured: s.appID != "" && s.appSecret != "",
		Connected:  isConnected,
		AppID:      s.appID,
	}
	if s.botName != "" {
		status.BotName = s.botName
	} else if status.Configured {
		status.BotName = "Feishu Bot"
	}
	return status, nil
}

// GetStoredToken retrieves the persisted OAuth token from SystemSetting.
func (s *OAuthService) GetStoredToken(ctx context.Context) (*StoredOAuthToken, error) {
	setting, err := s.repo.GetByKey(oauthTokenKey)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	if strings.TrimSpace(setting.Payload) == "" {
		return nil, nil
	}

	var token StoredOAuthToken
	if err := json.Unmarshal([]byte(setting.Payload), &token); err != nil {
		return nil, fmt.Errorf("feishu oauth: parse stored token: %w", err)
	}

	// Set BotName from stored token
	if token.BotName != "" {
		// Already populated from a previous exchange
	}
	return &token, nil
}

// ClearAuth removes the stored OAuth state and token.
func (s *OAuthService) ClearAuth(ctx context.Context) error {
	if err := s.repo.Upsert(&model.SystemSetting{
		Key:     oauthTokenKey,
		Payload: "",
	}); err != nil {
		return fmt.Errorf("feishu oauth: clear token: %w", err)
	}
	if err := s.repo.Upsert(&model.SystemSetting{
		Key:     oauthStateKey,
		Payload: "",
	}); err != nil {
		return err
	}
	s.botName = ""
	logger.InfoC("feishu", "OAuth auth cleared")
	return nil
}

// exchangeCode calls the Feishu OAuth token endpoint to exchange code for token.
func (s *OAuthService) exchangeCode(ctx context.Context, code string) (*StoredOAuthToken, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", s.appID)
	form.Set("client_secret", s.appSecret)
	form.Set("code", code)
	form.Set("redirect_uri", s.redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://open.feishu.cn/open-apis/authen/v1/oauth/token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Code         int    `json:"code"`
		Msg          string `json:"msg"`
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Name         string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("feishu api error (code=%d msg=%s)", result.Code, result.Msg)
	}

	now := time.Now()
	token := &StoredOAuthToken{
		TenantAccessToken: result.AccessToken,
		ExpiresAt:         now.Add(time.Duration(result.ExpiresIn) * time.Second).Unix(),
		AppID:             s.appID,
		AuthorizedAt:      now.Unix(),
		BotName:           result.Name,
	}

	logger.InfoCF("feishu", "OAuth code exchanged successfully", map[string]any{
		"expires_in": result.ExpiresIn,
		"bot_name":   result.Name,
	})
	return token, nil
}

// SeedTokenCache seeds a tokenCache with the stored OAuth token.
func (s *OAuthService) SeedTokenCache(ctx context.Context, tc *tokenCache) error {
	token, err := s.GetStoredToken(ctx)
	if err != nil {
		return err
	}
	if token == nil || token.TenantAccessToken == "" {
		return nil
	}

	remaining := time.Until(time.Unix(token.ExpiresAt, 0))
	if remaining < 0 {
		logger.InfoC("feishu", "stored OAuth token expired, will refresh on startup")
		return nil
	}

	// Restore bot name in memory
	if token.BotName != "" {
		s.botName = token.BotName
	}

	// tenant_access_token cache key is typically the app_id
	if err := tc.Set(ctx, token.AppID, token.TenantAccessToken, remaining); err != nil {
		return fmt.Errorf("seed token cache: %w", err)
	}

	logger.InfoCF("feishu", "seeded token cache from stored OAuth token", map[string]any{
		"remaining_seconds": int(remaining.Seconds()),
	})
	return nil
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
