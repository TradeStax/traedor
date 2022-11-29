package auth

import (
	"golang.org/x/oauth2"
	"net/http"
)

type HTTPHeaderStore struct{}

func (s *HTTPHeaderStore) StoreToken(token *oauth2.Token, w http.ResponseWriter, req *http.Request) error {
	// DO NOT DO THIS IN A PRODUCTION ENVIRONMENT!
	// This is just an example.
	// Used signed cookies like those provided by https://github.com/gorilla/securecookie
	http.SetCookie(
		w,
		&http.Cookie{
			Name:    "refreshToken",
			Value:   token.RefreshToken,
			Expires: token.Expiry,
		},
	)
	http.SetCookie(
		w,
		&http.Cookie{
			Name:    "accessToken",
			Value:   token.AccessToken,
			Expires: token.Expiry,
		},
	)
	return nil
}

func (s HTTPHeaderStore) GetToken(req *http.Request) (*oauth2.Token, error) {
	// DO NOT DO THIS IN A PRODUCTION ENVIRONMENT!
	// This is just an example.
	// Used signed cookies like those provided by https://github.com/gorilla/securecookie
	refreshToken, err := req.Cookie("refreshToken")
	if err != nil {
		return nil, err
	}

	accessToken, err := req.Cookie("accessToken")
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken:  accessToken.Value,
		RefreshToken: refreshToken.Value,
		Expiry:       refreshToken.Expires,
	}, nil
}

func (s HTTPHeaderStore) StoreState(state string, w http.ResponseWriter, req *http.Request) error {
	// DO NOT DO THIS IN A PRODUCTION ENVIRONMENT!
	// This is just an example.
	// Used signed cookies like those provided by https://github.com/gorilla/securecookie
	http.SetCookie(
		w,
		&http.Cookie{
			Name:  "state",
			Value: state,
		},
	)
	return nil
}

func (s HTTPHeaderStore) GetState(req *http.Request) (string, error) {
	// DO NOT DO THIS IN A PRODUCTION ENVIRONMENT!
	// This is just an example.
	// Used signed cookies like those provided by https://github.com/gorilla/securecookie
	cookie, err := req.Cookie("state")
	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}
