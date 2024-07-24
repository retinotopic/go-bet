package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/anandvarma/namegen"
	"github.com/google/uuid"
	"github.com/retinotopic/go-bet/pkg/randfuncs"
)

var secret = []byte(os.Getenv("SECRET_KEY"))

func WriteCookie(w http.ResponseWriter) *http.Cookie {
	mac := hmac.New(sha256.New, secret)
	value := uuid.New().String()
	ng := namegen.New()
	name := ng.GetForId(randfuncs.NewSource().Int63())
	cookie := &http.Cookie{Secure: true, Path: "/", HttpOnly: true}
	mac.Write([]byte(name))
	mac.Write([]byte(value))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	cookie.Value = value + signature
	cookie.Name = "guest" + name
	http.SetCookie(w, cookie)
	return cookie
}
func ReadCookie(r *http.Request) (string, error) {
	c, err := CookiesByPrefix(r, "guest")
	if err != nil {
		return "", err
	}
	name := c.Name
	valueHash := c.Value
	signature := valueHash[36:]
	value := valueHash[:36]

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(name))
	mac.Write([]byte(value))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return "", errors.New("ValidationErr")
	}
	return c.Name[5:], nil
}
func CookiesByPrefix(r *http.Request, prefix string) (*http.Cookie, error) {
	var matchingCookies []*http.Cookie
	cookieHeaders, ok := r.Header["Cookie"]
	if !ok || len(cookieHeaders) == 0 {
		return nil, http.ErrNoCookie
	}
	cookiePairs := strings.Split(cookieHeaders[0], ";")
	for _, cookiePair := range cookiePairs {
		cookiePair = strings.TrimSpace(cookiePair)
		if strings.HasPrefix(cookiePair, prefix) {
			parts := strings.SplitN(cookiePair, "=", 2)
			if len(parts) == 2 {
				cookie := &http.Cookie{
					Name:  parts[0],
					Value: parts[1],
				}
				matchingCookies = append(matchingCookies, cookie)
			}
		}
	}

	return matchingCookies[0], nil
}
