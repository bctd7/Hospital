package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultCode2SessionURL = "https://api.weixin.qq.com/sns/jscode2session"

type WeChatClient struct {
	appID     string
	appSecret string
	endpoint  string
	client    *http.Client
}

func NewWeChatClient(appID, appSecret, endpoint string, client *http.Client) *WeChatClient {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = DefaultCode2SessionURL
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &WeChatClient{
		appID: strings.TrimSpace(appID), appSecret: strings.TrimSpace(appSecret),
		endpoint: endpoint, client: client,
	}
}

func (c *WeChatClient) ExchangeLoginCode(ctx context.Context, code string) (WeChatSession, error) {
	if c.appID == "" || c.appSecret == "" {
		return WeChatSession{}, ErrProviderNotConfigured
	}
	if strings.TrimSpace(code) == "" {
		return WeChatSession{}, ErrInvalidCredential
	}
	endpoint, err := url.Parse(c.endpoint)
	if err != nil {
		return WeChatSession{}, fmt.Errorf("parse wechat code2session endpoint: %w", err)
	}
	query := endpoint.Query()
	query.Set("appid", c.appID)
	query.Set("secret", c.appSecret)
	query.Set("js_code", code)
	query.Set("grant_type", "authorization_code")
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return WeChatSession{}, fmt.Errorf("create wechat code2session request: %w", err)
	}
	response, err := c.client.Do(request)
	if err != nil {
		// url.Error includes the full request URL, which contains AppSecret and login code.
		return WeChatSession{}, fmt.Errorf("%w: code2session request failed", ErrProviderUnavailable)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return WeChatSession{}, fmt.Errorf("%w: code2session returned http status %d", ErrProviderUnavailable, response.StatusCode)
	}
	var payload struct {
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		SessionKey string `json:"session_key"`
		ErrorCode  int    `json:"errcode"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return WeChatSession{}, fmt.Errorf("decode wechat code2session response: %w", err)
	}
	if payload.ErrorCode != 0 || payload.OpenID == "" {
		return WeChatSession{}, ErrInvalidCredential
	}
	return WeChatSession{OpenID: payload.OpenID, UnionID: payload.UnionID, SessionKey: payload.SessionKey}, nil
}

func (c *WeChatClient) ExchangePhoneCode(context.Context, string) (WeChatPhone, error) {
	return WeChatPhone{}, errors.New("wechat phone verification is unavailable for the current miniapp subject")
}
