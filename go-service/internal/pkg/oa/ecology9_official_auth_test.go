package oa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"
)

func TestParseRSAPublicKeyAndEncrypt(t *testing.T) {
	// 1. 生成测试 RSA 密钥对
	privKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("生成测试密钥失败: %v", err)
	}

	// 2. 导出为 PKIX PEM 格式
	derPKIX, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("序列化公钥失败: %v", err)
	}
	pemPKIX := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPKIX,
	})

	// 测试解析 PEM 格式
	pubKey1, err := ParseRSAPublicKey(string(pemPKIX))
	if err != nil {
		t.Fatalf("解析 PEM 公钥失败: %v", err)
	}
	if pubKey1 == nil || pubKey1.N.Cmp(privKey.PublicKey.N) != 0 {
		t.Errorf("解析公钥不一致")
	}

	// 3. 测试解析裸 Base64 格式
	base64PKIX := base64.StdEncoding.EncodeToString(derPKIX)
	pubKey2, err := ParseRSAPublicKey(base64PKIX)
	if err != nil {
		t.Fatalf("解析裸 Base64 公钥失败: %v", err)
	}
	if pubKey2 == nil || pubKey2.N.Cmp(privKey.PublicKey.N) != 0 {
		t.Errorf("解析裸 Base64 公钥不一致")
	}

	// 4. 测试加密与解密验证
	testMessage := "secret_weaver_token_12345"
	encrypted, err := EncryptWithSPK(pubKey1, testMessage)
	if err != nil {
		t.Fatalf("公钥加密失败: %v", err)
	}

	cipherBytes, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("解密 base64 失败: %v", err)
	}

	decryptedBytes, err := rsa.DecryptPKCS1v15(rand.Reader, privKey, cipherBytes)
	if err != nil {
		t.Fatalf("私钥解密验证失败: %v", err)
	}

	if string(decryptedBytes) != testMessage {
		t.Errorf("加解密内容不符: 期望 %s, 实际 %s", testMessage, string(decryptedBytes))
	}
}
