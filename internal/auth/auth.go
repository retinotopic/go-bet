package auth

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/retinotopic/tinyauth/provider"
	"github.com/retinotopic/tinyauth/providers/firebase"
	"github.com/retinotopic/tinyauth/providers/google"
)

func init() {
	Mproviders.m = make(map[string]provider.Provider)
	google, err := google.New(os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), os.Getenv("REDIRECT"))
	if err != nil {
		log.Fatalln(err)
	}
	firebase, err := firebase.New(os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), os.Getenv("REDIRECT"))
	if err != nil {
		log.Fatalln(err)
	}
	Mproviders.m["firebase"] = firebase
	Mproviders.m["google"] = google
}

var Mproviders authprovider

type authprovider struct {
	m map[string]provider.Provider
}

func (a authprovider) GetProvider(w http.ResponseWriter, r *http.Request) (provider.Provider, error) {
	prvdr := r.URL.Query().Get("provider")
	if len(prvdr) == 0 {
		return nil, errors.New("malformed request")
	}
	if prvdr, ok := a.m[prvdr]; ok {
		return prvdr, nil
	}
	return nil, errors.New("no such provider")
}
