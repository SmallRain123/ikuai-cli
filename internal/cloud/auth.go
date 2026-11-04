// Package cloud 实现爱快开放平台 Open API 3.0 云端接入。
package cloud

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

// parsePublicKey 解析爱快开放平台 RSA 公钥，兼容 PKCS1 与 PKIX 两种 PEM 编码。
func parsePublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(pemStr)))
	if block == nil {
		return nil, fmt.Errorf("无法解析 RSA 公钥 PEM，请确认公钥格式")
	}

	switch block.Type {
	case "RSA PUBLIC KEY": // PKCS#1
		return x509.ParsePKCS1PublicKey(block.Bytes)
	case "PUBLIC KEY": // PKIX / SPKI
		anyKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("解析 PKIX 公钥失败: %w", err)
		}
		pub, ok := anyKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("公钥不是 RSA 类型")
		}
		return pub, nil
	default:
		return nil, fmt.Errorf("不支持的 PEM 类型: %s", block.Type)
	}
}

// rsaSign 计算 Open API 3.0 的签名：对 request_time 的字符串做 RSA 公钥加密后 base64。
// 参考官方文档：sign = base64_encode(rsa_enc_use_pubkey(request_time, pubkey))
func rsaSign(pub *rsa.PublicKey, requestTime int64) (string, error) {
	msg := []byte(fmt.Sprintf("%d", requestTime))
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, pub, msg)
	if err != nil {
		return "", fmt.Errorf("RSA 加密失败: %w", err)
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// tokenRequest 获取 AccessToken 的请求体。
type tokenRequest struct {
	ClientType  string           `json:"client_type"`
	ClientID    string           `json:"client_id"`
	RequestTime int64            `json:"request_time"`
	Options     tokenOptions     `json:"options"`
	Sign        string           `json:"sign"`
}

type tokenOptions struct {
	TokenLifetime int64 `json:"token_lifetime"`
}

type tokenRefreshRequest struct {
	AccessToken string      `json:"access_token"`
	ClientID    string      `json:"client_id"`
	Options     tokenOptions `json:"options"`
}

func nowUnix() int64 { return time.Now().Unix() }
