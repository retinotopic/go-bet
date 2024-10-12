package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"os"

	"github.com/google/uuid"
)

var secret = []byte(os.Getenv("SECRET_KEY"))

func WriteCookie(w http.ResponseWriter) *http.Cookie {
	mac := hmac.New(sha256.New, secret)
	str := uuid.New().String()
	cookie := &http.Cookie{Secure: true, Path: "/", HttpOnly: true}
	mac.Write([]byte(str))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	cookie.Value = str + signature
	cookie.Name = "guest"
	http.SetCookie(w, cookie)
	return cookie
}
func ReadCookie(r *http.Request) (string, error) {
	c, err := r.Cookie("guest")
	if err != nil {
		return "", err
	}
	valueHash := c.Value
	signature := valueHash[36:]
	value := valueHash[:36]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(value))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return "", errors.New("ValidationErr")
	}
	return c.Name[5:], nil
}
