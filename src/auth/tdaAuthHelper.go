package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/tradestax/go-tdameritrade"
	"github.com/tradestax/traedor/config"
	"golang.org/x/oauth2"
)

type TDAAuthHelper struct {
	authenticator   *tdameritrade.Authenticator
	authChan        chan error
	config          config.AuthConfig
	user            string
	token           string
	StreamingClient *tdameritrade.StreamingClient
	UPN             *tdameritrade.UserPrincipal
}

const (
	authURL       = "https://auth.tdameritrade.com/auth"
	tokenURL      = "https://api.tdameritrade.com/v1/oauth2/token"
	serverAddress = ":8081"
)

func NewTDAAuthHelper(c config.Config) *TDAAuthHelper {
	clientID := os.Getenv(c.AuthConfig.UserEnvVar)
	if clientID == "" {
		panic(fmt.Errorf("Unauthorized: No client ID present"))
	}
	return &TDAAuthHelper{
		authChan: make(chan error, 1),
		config:   c.AuthConfig,
		user:     clientID,
	}
}

func (a *TDAAuthHelper) Authenticate() error {
	a.authenticator = tdameritrade.NewAuthenticator(
		&HTTPHeaderStore{},
		oauth2.Config{
			ClientID: a.user,
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenURL,
				AuthURL:  authURL,
			},
			RedirectURL: a.config.CallbackURL,
		},
	)
	http.HandleFunc("/authenticate", a.AuthenticateHandler)
	http.HandleFunc("/callback", a.CallbackHandler)
	http.HandleFunc("/stream", a.StreamHandler)
	srv := &http.Server{Addr: serverAddress}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		fmt.Printf("Listening on %v\n", serverAddress)
		fmt.Printf("Please visit %v/authenticate to authorize application\n", serverAddress)
		a.authChan <- srv.ListenAndServe()
		fmt.Println("Auth complete http server shutting down...")
		wg.Done()
	}()
	err := <-a.authChan
	srv.Shutdown(context.Background())
	wg.Wait()
	return err
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

func (h *TDAAuthHelper) AuthenticateHandler(w http.ResponseWriter, req *http.Request) {
	redirectURL, err := h.authenticator.StartOAuth2Flow(w, req)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, redirectURL, http.StatusTemporaryRedirect)
}

func (h *TDAAuthHelper) CallbackHandler(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	_, err := h.authenticator.FinishOAuth2Flow(ctx, w, req)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, "/stream", http.StatusFound)
}

func (h *TDAAuthHelper) StreamHandler(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	client, err := h.authenticator.AuthenticatedClient(ctx, req)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		h.authChan <- err
		return
	}
	userPrincipals, resp, err := client.User.GetUserPrincipals(ctx, "streamerSubscriptionKeys", "streamerConnectionInfo")
	if err != nil {
		h.authChan <- err
		log.Fatal(err)
	}
	log.Printf("User Principals: [Status: %d] %+v", resp.StatusCode, *userPrincipals)
	h.UPN = userPrincipals
	h.StreamingClient, err = tdameritrade.NewAuthenticatedStreamingClient(userPrincipals, userPrincipals.Accounts[0].AccountID)
	if err != nil {
		h.authChan <- err
		log.Fatal(err)
	}
	w.Write([]byte("Successfully Authenticated, you can now close this page"))
	w.WriteHeader(http.StatusOK)
	h.authChan <- nil
}
