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
		fmt.Println("No auth helper specified, returning empty Interface")
	}
	return a
}
