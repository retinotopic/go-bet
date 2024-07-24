package auth

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt"
	"github.com/retinotopic/tinyauth/provider"
	"github.com/retinotopic/tinyauth/providers/firebase"
	"github.com/retinotopic/tinyauth/providers/google"
)

func init() {
	Mproviders.m = make(map[string]provider.Provider)
	google, err := google.New(os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), os.Getenv("REDIRECT"), "/refresh")
	if err != nil {
		log.Fatalln(err)
	}
	firebase, err := firebase.New(os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), os.Getenv("REDIRECT"), "refresh")
	if err != nil {
		log.Fatalln(err)
	}

	// with the issuer claim, we know which provider issued token.
	Mproviders.m[os.Getenv("ISSUER_FIREBASE")] = firebase
	Mproviders.m[os.Getenv("ISSUER_GOOGLE")] = google
	Mproviders.m[os.Getenv("ISSUER2_GOOGLE")] = google
}

var Mproviders authprovider

type authprovider struct {
	m map[string]provider.Provider
}

func (a authprovider) GetProvider(w http.ResponseWriter, r *http.Request) (provider.Provider, error) {
	cookie, err := r.Cookie("token")
	if err != nil {
		return nil, err
	}
	token, _, err := new(jwt.Parser).ParseUnverified(cookie.Value, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}
	var issuer string
	var ok bool
	var claims jwt.MapClaims
	var iss interface{}
	if claims, ok = token.Claims.(jwt.MapClaims); !ok {
		return nil, fmt.Errorf("issuer claim retrieve error")
	}
	if iss, ok = claims["iss"]; !ok {
		return nil, fmt.Errorf("issuer claim retrieve error")
	}
	if issuer, ok = iss.(string); !ok {
		return nil, fmt.Errorf("issuer claim is not a string")
	}

	if prvdr, ok := a.m[issuer]; ok {
		return prvdr, nil
	}
	return nil, errors.New("no such provider")
}
