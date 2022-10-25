package auth

import (
	"github.com/tradestax/go-tdameritrade"
	"github.com/tradestax/traedor/config"
)

type TDAAuthHelper struct {
	authenticator *tdameritrade.Authenticator
	user          string
	token         string
}

func NewTDAAuthHelper(c config.Config) *TDAAuthHelper {
	return &TDAAuthHelper{}
}

func (a *TDAAuthHelper) Authenticate() error {
	return nil
}

func (a *TDAAuthHelper) SetToken(t string) {
	a.token = t
}

func (a *TDAAuthHelper) SetUser(u string) {
	a.user = u
}

func (a *TDAAuthHelper) User() string {
	return a.user
}

func (a *TDAAuthHelper) Token() string {
	return a.token
}
