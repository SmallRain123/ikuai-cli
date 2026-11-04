package cloud

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ikuai-cli/internal/model"
)

// Client 云端开放平台 API 客户端。
type Client struct {
	apiBase  string
	openID   string
	pubKey   *rsa.PublicKey
	http     *http.Client
	token    string
	tokenExp time.Time
	lifetime int64
}

// apiResponse 开放平台统一响应包装。
type apiResponse struct {
	ErrorCode int64           `json:"ErrorCode"`
	ErrorMsg  string          `json:"ErrorMsg"`
	Data      json.RawMessage `json:"Data"`
}

// New 创建云端客户端。openID 为开放平台 open_id；pubKeyPEM 为 RSA 公钥 PEM 内容。
func New(openID, pubKeyPEM, apiBase string, lifetime int64) (*Client, error) {
	pub, err := parsePublicKey(pubKeyPEM)
	if err != nil {
		return nil, err
	}
	if apiBase == "" {
		apiBase = "https://open.ikuai8.com/api/v3"
	}
	if lifetime <= 0 {
		lifetime = 7200
	}
	return &Client{
		apiBase:  strings.TrimRight(apiBase, "/"),
		openID:   openID,
		pubKey:   pub,
		http:     &http.Client{Timeout: 30 * time.Second},
		lifetime: lifetime,
	}, nil
}

// GetToken 获取 AccessToken 并缓存。
func (c *Client) GetToken() error {
	reqTime := nowUnix()
	sign, err := rsaSign(c.pubKey, reqTime)
	if err != nil {
		return err
	}
	body := tokenRequest{
		ClientType:  "OPEN_PARTNER",
		ClientID:    c.openID,
		RequestTime: reqTime,
		Options:     tokenOptions{TokenLifetime: c.lifetime},
		Sign:        sign,
	}
	var resp model.TokenAccess
	if err := c.post("/open/access_token", "", body, &resp); err != nil {
		return err
	}
	c.token = resp.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(resp.TokenExcessTime) * time.Second)
	return nil
}

// RefreshToken 使用现有 Token 刷新。
func (c *Client) RefreshToken() error {
	body := tokenRefreshRequest{
		AccessToken: c.token,
		ClientID:    c.openID,
		Options:     tokenOptions{TokenLifetime: c.lifetime},
	}
	var resp model.TokenAccess
	if err := c.post("/open/refresh_token", "", body, &resp); err != nil {
		return err
	}
	c.token = resp.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(resp.TokenExcessTime) * time.Second)
	return nil
}

// ensureToken 确保 Token 有效，过期则自动刷新。
func (c *Client) ensureToken() error {
	if c.token == "" {
		return c.GetToken()
	}
	if time.Now().After(c.tokenExp.Add(-60 * time.Second)) {
		if err := c.RefreshToken(); err != nil {
			return c.GetToken() // 刷新失败则重新获取
		}
	}
	return nil
}

// post 发送 POST JSON 请求；body 为 nil 表示空 body。
func (c *Client) post(path string, gwid string, reqBody, out any) error {
	var payload io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(data)
	}

	url := c.apiBase + path
	if gwid != "" {
		url += "/" + gwid
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, payload)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if path != "/open/access_token" && path != "/open/refresh_token" {
		if err := c.ensureToken(); err != nil {
			return err
		}
		httpReq.Header.Set("IK-Access-Token", c.token)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("请求 %s 失败: %w", url, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == 401 {
		// Token 失效，重取一次
		if err := c.GetToken(); err != nil {
			return err
		}
		httpReq.Header.Set("IK-Access-Token", c.token)
		resp2, err2 := c.http.Do(httpReq)
		if err2 != nil {
			return err2
		}
		defer resp2.Body.Close()
		raw, err = io.ReadAll(resp2.Body)
		if err != nil {
			return err
		}
		resp = resp2
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var wrapper apiResponse
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return fmt.Errorf("解析响应失败: %w, body=%s", err, string(raw))
	}
	if wrapper.ErrorCode != 0 && wrapper.ErrorCode != 3000 {
		return fmt.Errorf("接口错误 [%d]: %s", wrapper.ErrorCode, wrapper.ErrorMsg)
	}

	if out == nil || len(wrapper.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(wrapper.Data, out); err != nil {
		return fmt.Errorf("解析 Data 失败: %w, body=%s", err, string(raw))
	}
	return nil
}

// Token 返回当前缓存的 AccessToken。
func (c *Client) Token() string { return c.token }

// TokenExpireAt 返回 Token 过期时间（格式化字符串）。
func (c *Client) TokenExpireAt() string {
	if c.tokenExp.IsZero() {
		return ""
	}
	return c.tokenExp.Local().Format("2006-01-02 15:04:05")
}

// DeviceList 获取授权设备列表。
func (c *Client) DeviceList() ([]model.Device, error) {
	var list []model.Device
	if err := c.post("/open/dev_list", "", nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// SystemStatus 获取路由器基本信息与实时流量。
func (c *Client) SystemStatus(gwid string) (*model.SystemStatus, error) {
	var st model.SystemStatus
	if err := c.post("/open/9", gwid, nil, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// OnlineUsers 获取在线用户列表。limit<=0 时使用 50。
func (c *Client) OnlineUsers(gwid string, limit, skip int) ([]model.OnlineUser, error) {
	if limit <= 0 {
		limit = 50
	}
	body := map[string]any{"param": map[string]any{
		"TYPE": "data", "skip": skip, "limit": limit,
	}}
	var list []model.OnlineUser
	if err := c.post("/open/4", gwid, body, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// Accounts 获取认证账号列表。
func (c *Client) Accounts(gwid string) ([]model.Account, error) {
	body := map[string]any{"param": map[string]any{"TYPE": "data"}}
	var list []model.Account
	if err := c.post("/open/1", gwid, body, &list); err != nil {
		return nil, err
	}
	return list, nil
}
