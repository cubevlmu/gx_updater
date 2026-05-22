package security

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

func RandomBase64(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

func GenerateEd25519KeyPair() (publicKeyBase64 string, privateKeyBase64 string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(pub), base64.StdEncoding.EncodeToString(priv), nil
}

func HMACKey(secret string) []byte {
	if raw, err := base64.StdEncoding.DecodeString(secret); err == nil && len(raw) > 0 {
		return raw
	}
	return []byte(secret)
}

func BodySHA256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func RequestSigningPayload(method, path, timestamp, nonce string, body []byte) string {
	return strings.ToUpper(method) + "\n" + path + "\n" + timestamp + "\n" + nonce + "\n" + BodySHA256Hex(body)
}

func SignRequestHMAC(secret, method, path, timestamp, nonce string, body []byte) string {
	mac := hmac.New(sha256.New, HMACKey(secret))
	_, _ = mac.Write([]byte(RequestSigningPayload(method, path, timestamp, nonce, body)))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func VerifyRequestHMAC(secret, method, path, timestamp, nonce string, body []byte, signature string) bool {
	expected := SignRequestHMAC(secret, method, path, timestamp, nonce, body)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) == 1
}

func SHA256FileHex(r io.Reader) (string, int64, error) {
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

type ManifestPayload struct {
	AppID            string
	Version          string
	VersionCode      int64
	Platform         string
	Arch             string
	Channel          string
	PackageURL       string
	PackageSize      int64
	PackageSHA256    string
	Force            bool
	MinSupportedCode int64
}

func CanonicalManifestPayload(m ManifestPayload) string {
	return fmt.Sprintf(
		"gx-update-manifest-v1\napp_id=%s\nversion=%s\nversion_code=%d\nplatform=%s\narch=%s\nchannel=%s\npackage_url=%s\npackage_size=%d\npackage_sha256=%s\nforce=%t\nmin_supported_code=%d\n",
		m.AppID,
		m.Version,
		m.VersionCode,
		m.Platform,
		m.Arch,
		m.Channel,
		m.PackageURL,
		m.PackageSize,
		m.PackageSHA256,
		m.Force,
		m.MinSupportedCode,
	)
}

func SignManifest(privateKeyBase64 string, payload ManifestPayload) (string, error) {
	priv, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return "", err
	}
	if len(priv) != ed25519.PrivateKeySize {
		return "", errors.New("invalid ed25519 private key size")
	}
	sig := ed25519.Sign(ed25519.PrivateKey(priv), []byte(CanonicalManifestPayload(payload)))
	return base64.StdEncoding.EncodeToString(sig), nil
}

func VerifyManifest(publicKeyBase64 string, payload ManifestPayload, signatureBase64 string) (bool, error) {
	pub, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return false, err
	}
	if len(pub) != ed25519.PublicKeySize {
		return false, errors.New("invalid ed25519 public key size")
	}
	sig, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return false, err
	}
	return ed25519.Verify(ed25519.PublicKey(pub), []byte(CanonicalManifestPayload(payload)), sig), nil
}

func IssueDownloadToken(secret string, packageID uint, clientID string, ttl time.Duration) string {
	_ = clientID // kept in signature for API compatibility; token itself is package-scoped and short-lived.
	expires := time.Now().Add(ttl).Unix()
	payload := fmt.Sprintf("%d:%d", packageID, expires)
	mac := hmac.New(sha256.New, HMACKey(secret))
	_, _ = mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload + ":" + sig))
}

func VerifyDownloadToken(secret string, packageID uint, clientID, token string) error {
	_ = clientID
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return err
	}
	parts := strings.Split(string(raw), ":")
	if len(parts) != 3 {
		return errors.New("invalid token format")
	}
	var id uint
	var expires int64
	if _, err := fmt.Sscanf(parts[0], "%d", &id); err != nil {
		return err
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &expires); err != nil {
		return err
	}
	if id != packageID {
		return errors.New("package id mismatch")
	}
	if time.Now().Unix() > expires {
		return errors.New("token expired")
	}
	payload := strings.Join(parts[:2], ":")
	mac := hmac.New(sha256.New, HMACKey(secret))
	_, _ = mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[2])) != 1 {
		return errors.New("bad token signature")
	}
	return nil
}

func StablePercent(key string) int {
	sum := sha256.Sum256([]byte(key))
	v := int(sum[0])<<8 | int(sum[1])
	return v % 100
}
