package auth

type IAuthHelper interface {
	Authenticate() error
	SetToken(string)
	SetUser(string)
	User() string
	Token() string
}
