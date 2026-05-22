package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func main() {
	secret := "<client_secret_from_create_project>"
	body := []byte(`{"app_id":"com.example.class_tool","platform":"android","arch":"universal","channel":"stable","version":"1.0.0","version_code":100,"client_id":"demo"}`)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := "demo-nonce-uuid"
	path := "/api/v1/check"

	signature := signRequest(secret, "POST", path, timestamp, nonce, body)

	req, _ := http.NewRequest("POST", "http://127.0.0.1:8080"+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-App-Id", "com.example.class_tool")
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", signature)

	fmt.Println("X-Signature:", signature)
}

func signRequest(secret, method, path, timestamp, nonce string, body []byte) string {
	key, err := base64.StdEncoding.DecodeString(secret)
	if err != nil || len(key) == 0 {
		key = []byte(secret)
	}
	bodySum := sha256.Sum256(body)
	payload := strings.ToUpper(method) + "\n" + path + "\n" + timestamp + "\n" + nonce + "\n" + hex.EncodeToString(bodySum[:])
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(payload))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
