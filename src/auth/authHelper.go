package auth

import (
	"fmt"

	"github.com/tradestax/traedor/config"
)

func NewAuthHelper(c *config.Config) IAuthHelper {
	var a IAuthHelper
	switch c.AuthConfig.AuthHelper {
	case "TDA":
		a = NewTDAAuthHelper(c)
	default:
		fmt.Println("No auth helper specified, returning default auth helper")
		a = &AuthHelper{}
	}
	return a
}

type AuthHelper struct {
	token string
	user  string
}

func (a *AuthHelper) Authenticate() error {
	return nil
}

func (a *AuthHelper) SetToken(t string) {
	a.token = t
}

func (a *AuthHelper) SetUser(u string) {
	a.user = u
}

func (a *AuthHelper) User() string {
	return a.user
}

func (a *AuthHelper) Token() string {
	return a.token
}
