package cloud

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"testing"
)

// genPEM 生成测试用 RSA 密钥对，返回 PKCS1/PKIX 两种 PEM 公钥及私钥。
func genPEM(t *testing.T) (pkcs1, pkix string, priv *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	pkcs1 = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&key.PublicKey)}))
	pkixBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal pkix: %v", err)
	}
	pkix = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkixBytes}))
	return pkcs1, pkix, key
}

func TestParsePublicKeyAndSign(t *testing.T) {
	pkcs1, pkix, priv := genPEM(t)
	for name, pemStr := range map[string]string{"PKCS1": pkcs1, "PKIX": pkix} {
		pub, err := parsePublicKey(pemStr)
		if err != nil {
			t.Fatalf("[%s] parse failed: %v", name, err)
		}
		reqTime := int64(1503525563)
		sign, err := rsaSign(pub, reqTime)
		if err != nil {
			t.Fatalf("[%s] sign failed: %v", name, err)
		}
		raw, err := base64.StdEncoding.DecodeString(sign)
		if err != nil {
			t.Fatalf("[%s] base64 decode: %v", name, err)
		}
		decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, priv, raw)
		if err != nil {
			t.Fatalf("[%s] decrypt: %v", name, err)
		}
		want := fmt.Sprintf("%d", reqTime)
		if string(decrypted) != want {
			t.Fatalf("[%s] sign mismatch: got %q want %q", name, string(decrypted), want)
		}
	}
}

func TestParsePublicKeyInvalid(t *testing.T) {
	if _, err := parsePublicKey("not a pem"); err == nil {
		t.Fatal("expected error for invalid pem")
	}
	if _, err := parsePublicKey(`-----BEGIN PRIVATE KEY-----
MIIE
-----END PRIVATE KEY-----`); err == nil {
		t.Fatal("expected error for private key block")
	}
}
