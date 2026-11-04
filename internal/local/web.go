// Package local 实现爱快路由器的本地 Web 管理接口接入（实验性）。
// 注意：本地 Web 接口为非官方接口，端点和登录算法随路由固件版本变化，
// 本包提供登录层 + 通用调用（Call），并内置一组可校准的预设查询。
package local

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"
)

// 预设查询端点（默认值，不同版本路由可能不同，可用 --endpoint 覆盖）。
const (
	DefaultLogEndpoint = "/api/log/list"
	DefaultConnEndpoint = "/api/conntrack/table"
	LoginPath          = "/Action/login"
)

// WebClient 本地 Web 管理接口客户端。
type WebClient struct {
	baseURL   string
	user      string
	pass      string
	loginMode string // md5 / plain / auto
	http      *http.Client
	csrf      string
	loggedIn  bool
}

// NewWebClient 创建本地 Web 客户端。baseURL 形如 http://192.168.9.1 或 https://192.168.9.1。
// insecure 为 true 时忽略 HTTPS 证书校验（适用于内网自签证书）。
func NewWebClient(baseURL, user, pass, loginMode string, insecure bool) (*WebClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	if loginMode == "" {
		loginMode = "auto"
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}, // #nosec G402 内网自签证书场景
	}
	return &WebClient{
		baseURL:   strings.TrimRight(baseURL, "/"),
		user:      user,
		pass:      pass,
		loginMode: loginMode,
		http: &http.Client{
			Jar:       jar,
			Transport: tr,
			Timeout:   20 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil // 允许跳转但保留cookie
			},
		},
	}, nil
}

// Login 登录爱快本地 Web 管理界面。
func (w *WebClient) Login() error {
	// 1) 访问首页获取 csrf token 与 cookie
	if err := w.fetchCSRF(); err != nil {
		return err
	}

	// 2) 构造登录请求，密码按模式处理
	passCandidates := w.passwordCandidates()

	var lastErr error
	for _, pass := range passCandidates {
		body, _ := json.Marshal(map[string]string{
			"username": w.user,
			"passwd":   w.pass,
			"pass":     pass,
		})
		req, err := http.NewRequest(http.MethodPost, w.baseURL+LoginPath, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		if w.csrf != "" {
			req.Header.Set("X-CSRFToken", w.csrf)
		}
		resp, err := w.http.Do(req)
		if err != nil {
			return fmt.Errorf("登录请求失败: %w", err)
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if loginOK(raw) {
			w.loggedIn = true
			return nil
		}
		lastErr = fmt.Errorf("登录被拒(HTTP %d, pass=%s): %s", resp.StatusCode, short(pass), string(raw))
	}
	return fmt.Errorf("本地 Web 登录失败: %w", lastErr)
}

// passwordCandidates 按登录模式生成候选密码。
func (w *WebClient) passwordCandidates() []string {
	md := md5sum(w.pass)
	switch w.loginMode {
	case "md5":
		return []string{md}
	case "plain":
		return []string{w.pass}
	default: // auto：先 md5，再明文
		return []string{md, w.pass}
	}
}

// fetchCSRF 获取登录页并尝试提取 csrf token。
func (w *WebClient) fetchCSRF() error {
	resp, err := w.http.Get(w.baseURL + "/")
	if err != nil {
		return fmt.Errorf("访问登录页失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	s := string(raw)
	// 常见 csrf 提取：meta[name=csrf] content、csrfmiddlewaretoken、csrftoken
	re := regexp.MustCompile(`(?i)csrf.{0,40}?[\"']?([a-zA-Z0-9]{20,64})[\"']?`)
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		w.csrf = m[1]
	}
	return nil
}

// Call 以登录态调用本地 Web 接口，返回原始响应字节。
func (w *WebClient) Call(method, path string, body any) ([]byte, error) {
	if !w.loggedIn {
		if err := w.Login(); err != nil {
			return nil, err
		}
	}
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		payload = bytes.NewReader(data)
	}
	url := w.baseURL + strings.TrimPrefix(path, "/")
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if w.csrf != "" {
		req.Header.Set("X-CSRFToken", w.csrf)
	}
	resp, err := w.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用 %s 失败: %w", path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("调用 %s 返回 HTTP %d: %s", path, resp.StatusCode, string(raw))
	}
	return raw, nil
}

// LoggedIn 返回是否已登录。
func (w *WebClient) LoggedIn() bool { return w.loggedIn }

// loginOK 简单判断登录是否成功：errno==0 且 errmsg 含 success。
func loginOK(raw []byte) bool {
	var v struct {
		Errno  int    `json:"errno"`
		Errmsg string `json:"errmsg"`
		ErrorCode int `json:"ErrorCode"`
	}
	if err := json.Unmarshal(raw, &v); err == nil {
		if v.Errno == 0 && strings.Contains(strings.ToLower(v.Errmsg), "success") {
			return true
		}
		if v.ErrorCode == 0 && strings.Contains(strings.ToLower(v.Errmsg), "success") {
			return true
		}
	}
	s := strings.ToLower(string(raw))
	return strings.Contains(s, "success") || strings.Contains(s, "\"errno\":0")
}

func md5sum(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func short(s string) string {
	if len(s) <= 6 {
		return s
	}
	return s[:6] + "..."
}
