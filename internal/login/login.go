// Package login defines the authentication subsystem of iblcli. This core lib
// can be used by other packages to authenticate with the IBL API.
package login

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/InfinityBotList/ibl/internal/api"
	"github.com/InfinityBotList/ibl/internal/config"
	"github.com/InfinityBotList/ibl/internal/input"
	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/types"
	"github.com/InfinityBotList/ibl/types/popltypes"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/infinitybotlist/eureka/uapi"
)

type iLogin struct {
	code  string
	state string
	err   error
}

var loginCh = make(chan iLogin)

// WebAuthUser performs a web-based OAuth2 login for users
func WebAuthUser() (string, string, error) {
	// Load login webserver
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/sauron", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		w.Write([]byte("You can now close this window"))

		loginCh <- iLogin{
			code:  code,
			state: state,
		}
	})

	resp, err := api.NewReq().Get("list/oauth2").Do()

	if err != nil {
		return "", "", errors.New("error occurred while getting OAuth2 URL: " + err.Error())
	}

	var oauth2Data popltypes.OauthMeta

	err = resp.Json(&oauth2Data)

	if err != nil {
		fmt.Print(ui.RedText("Error parsing OAuth2 URL: " + err.Error()))
		return "", "", err
	}

	// Open a http server on port 3000
	srv := &http.Server{Addr: ":3000", Handler: mux}

	go func() {
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			loginCh <- iLogin{
				code:  "",
				state: "",
				err:   err,
			}
			return
		}
	}()

	state := crypto.RandString(32)

	fmt.Println("")
	fmt.Println("")
	fmt.Print(ui.BlueText("Please open the following URL in your browser and follow the instructions:"))
	fmt.Println("")
	fmt.Println(strings.ReplaceAll(oauth2Data.URL, "%REDIRECT_URL%", "http://localhost:3000") + "&state=" + state)

	// Wait for login
	login := <-loginCh

	if login.err != nil {
		time.Sleep(1 * time.Second)
		return "", "", errors.New("error occurred while waiting for login: " + login.err.Error())
	}

	if os.Getenv("DEBUG") == "true" {
		fmt.Println("Logging in: code="+login.code, "| state="+login.state)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

	srv.Shutdown(ctx)

	cancel()

	if login.state != state {
		time.Sleep(5 * time.Second)
		return "", "", errors.New("invalid state, please try again")
	}

	// Exchange code for token
	resp, err = api.NewReq().Put("users").Json(popltypes.AuthorizeRequest{
		ClientID:    oauth2Data.ClientID,
		Code:        login.code,
		Scope:       "external_auth",
		Protocol:    "persepolis",
		RedirectURI: "http://localhost:3000/auth/sauron",
	}).Do()

	if err != nil {
		time.Sleep(5 * time.Second)
		return "", "", errors.New("error occurred while exchanging code for token: " + err.Error())
	}

	if resp.Response.StatusCode != 200 {
		fmt.Println("Login failed, got response code", resp.Response.StatusCode)

		body, err := resp.Body()

		if err != nil {
			return "", "", errors.New("error occurred while parsing error when exchanging code for token: " + err.Error())
		}

		fmt.Println("Error body:", string(body))
		return "", "", errors.New("login failed, got response code " + fmt.Sprint(resp.Response.StatusCode))
	}

	var loginData popltypes.UserLogin

	err = resp.Json(&loginData)

	if err != nil {
		return "", "", errors.New("error occurred while parsing login data: " + err.Error())
	}

	return loginData.UserID, loginData.Token, nil
}

// LoginUser performs a login for the user/bot/server
func LoginUser(authType string) (*popltypes.TestAuth, error) {
	fmt.Print(ui.BoldBlueText(ui.AddUnderDecor("Login")))

	if strings.ToLower(authType) != "bot" && strings.ToLower(authType) != "user" && strings.ToLower(authType) != "server" {
		authType = input.GetInput("Auth Type (bot/user/server)", func(s string) bool {
			if strings.ToLower(s) == "bot" || strings.ToLower(s) == "user" || strings.ToLower(s) == "server" {
				return true
			} else {
				fmt.Fprintln(os.Stderr, "Invalid auth type. Choose from bot, user or server")
				return false
			}
		})
	}

	var targetType types.TargetType

	switch strings.ToLower(authType) {
	case "bot":
		targetType = types.TargetTypeBot
	case "user":
		targetType = types.TargetTypeUser
	case "server":
		targetType = types.TargetTypeServer
	default:
		return nil, errors.New("invalid auth type")
	}

	var targetID string
	var token string
	if targetType == types.TargetTypeUser {
		var webAuth = input.GetInput("Do you have a working browser for web auth right now? If not, type 'no' to use standard token auth. Headless/server users should also type 'no' here. [yes/no]", func(s string) bool {
			return s == "yes" || s == "no" || s == "y" || s == "n"
		})

		if webAuth == "yes" || webAuth == "y" {
			// Create external auth
			var err error
			targetID, token, err = WebAuthUser()

			if err != nil {
				return nil, errors.New("error occurred while performing web auth: " + err.Error())
			}
		}
	}

	if len(targetID) == 0 {
		targetID = input.GetInput("Target ID ["+authType+" ID, vanities are also supported]", func(s string) bool {
			return len(s) > 0
		})

		token = input.GetPassword("API Token [you can get this from bot/profile/server settings]")
	}

	// Check auth with API
	resp, err := api.NewReq().Post("list/auth-test").Json(popltypes.TestAuth{
		AuthType: string(targetType),
		TargetID: targetID,
		Token:    token,
	}).Do()

	if err != nil {
		return nil, errors.New("error occurred while validating auth: " + err.Error())
	}

	if resp.Response.StatusCode != 200 {
		return nil, errors.New("invalid token, got response code " + fmt.Sprint(resp.Response.StatusCode))
	}

	var payload uapi.AuthData
	err = resp.Json(&payload)

	if err != nil {
		return nil, errors.New("error occurred while parsing auth data: " + err.Error())
	}

	if os.Getenv("DEBUG") == "true" {
		fmt.Println("Server Response:", payload)
	}

	// Write the config
	auth := popltypes.TestAuth{
		AuthType: payload.TargetType,
		TargetID: payload.ID,
		Token:    token,
	}

	err = config.WriteConfig("auth@"+string(payload.TargetType), auth)

	if err != nil {
		return nil, errors.New("error occurred while writing config: " + err.Error())
	}

	return &auth, nil
}
