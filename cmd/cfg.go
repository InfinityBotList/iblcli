/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"net/http"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/InfinityBotList/ibl/types"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/spf13/cobra"
)

type iLogin struct {
	code  string
	state string
}

var loginCh = make(chan iLogin)

func init() {
	// Load login webserver
	http.HandleFunc("/auth/sauron", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		w.Write([]byte("You can now close this window"))

		loginCh <- iLogin{
			code:  code,
			state: state,
		}
	})
}

func webAuthUser() (string, string, error) {
	resp, err := helpers.NewReq().Get("list/oauth2").Do()

	if err != nil {
		return "", "", errors.New("error occurred while getting OAuth2 URL: " + err.Error())
	}

	var oauth2Data types.OauthMeta

	err = resp.Json(&oauth2Data)

	if err != nil {
		fmt.Print(helpers.RedText("Error parsing OAuth2 URL: " + err.Error()))
		return "", "", err
	}

	// Open a http server on port 3000
	srv := &http.Server{Addr: ":3000"}

	go func() {
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	state := crypto.RandString(32)

	fmt.Println("")
	fmt.Println("")
	fmt.Print(helpers.BlueText("Please open the following URL in your browser and follow the instructions:"))
	fmt.Println("")
	fmt.Println(strings.ReplaceAll(oauth2Data.URL, "%REDIRECT_URL%", "http://localhost:3000") + "&state=" + state)

	// Wait for login
	login := <-loginCh

	fmt.Println("Got login code", login.code, "with state", login.state)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

	srv.Shutdown(ctx)

	cancel()

	if login.state != state {
		time.Sleep(5 * time.Second)
		return "", "", errors.New("invalid state, please try again")
	}

	// Exchange code for token
	resp, err = helpers.NewReq().Put("users").Json(types.AuthorizeRequest{
		ClientID:    oauth2Data.ClientID,
		Code:        login.code,
		Scope:       "external_auth",
		Nonce:       "@external",
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

	var loginData types.UserLogin

	err = resp.Json(&loginData)

	if err != nil {
		return "", "", errors.New("error occurred while parsing login data: " + err.Error())
	}

	return loginData.UserID, loginData.Token, nil
}

// cfgCmd represents the cfg command
var cfgCmd = &cobra.Command{
	Use:   "cfg",
	Short: "Configuration system",
	Long:  `Configure the "ibl" client (authentication and other settings).`,
}

var loginCmd = &cobra.Command{
	Use:     "login",
	Short:   "Login to the IBL API",
	Long:    `Login to the IBL API using a bot/user/server token.`,
	Aliases: []string{"auth", "a", "l"},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(helpers.BoldBlueText(helpers.AddUnderDecor("Login")))

		var authType = os.Getenv("REQUIRED_AUTH_METHOD")

		if strings.ToLower(authType) != "bot" && strings.ToLower(authType) != "user" && strings.ToLower(authType) != "server" {
			authType = helpers.GetInput("Auth Type (bot/user/server)", func(s string) bool {
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
			fmt.Println("Invalid auth type")
			os.Exit(1)
		}

		var targetID string
		var token string
		if targetType == types.TargetTypeUser {
			var webAuth = helpers.GetInput("Do you have a working browser for web auth right now? If not, type 'no' to use standard token auth. Headless/server users should also type 'no' here", func(s string) bool {
				return s == "yes" || s == "no"
			})

			if webAuth == "yes" {
				// Create external auth
				var err error
				targetID, token, err = webAuthUser()

				if err != nil {
					fmt.Print(helpers.RedText("ERROR: " + err.Error()))
					os.Exit(1)
				}
			}
		}

		if len(targetID) == 0 {
			targetID = helpers.GetInput("Target ID ["+authType+" ID, vanities are also supported]", func(s string) bool {
				return len(s) > 0
			})

			token = helpers.GetPassword("API Token [you can get this from bot/profile/server settings]")
		}

		// Check auth with API
		resp, err := helpers.NewReq().Post("list/auth-test").Json(types.TestAuth{
			AuthType: targetType,
			TargetID: targetID,
			Token:    token,
		}).Do()

		if err != nil {
			fmt.Println("Error logging in:", err)
			os.Exit(1)
		}

		if resp.Response.StatusCode != 200 {
			fmt.Println("Invalid token, got response code", resp.Response.StatusCode)
			os.Exit(1)
		}

		var payload types.AuthData
		err = resp.Json(&payload)

		if err != nil {
			fmt.Println("Error logging in:", err)
			os.Exit(1)
		}

		fmt.Println("Server Response:", payload)

		// Write the config
		err = helpers.WriteConfig("auth@"+string(payload.TargetType), types.TestAuth{
			AuthType: payload.TargetType,
			TargetID: payload.ID,
			Token:    token,
		})

		if err != nil {
			fmt.Println("Error writing config:", err)
			os.Exit(1)
		}
	},
}

var setWebhookSecretCmd = &cobra.Command{
	Use:     "setwebhooksecret",
	Short:   "Set the webhook secret",
	Long:    `Set the webhook secret for the currently logged in bot.`,
	Aliases: []string{"websecret"},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(helpers.BoldBlueText(helpers.AddUnderDecor("Set Webhook Secret")))

		cfg, ok := helpers.LoadConfig("auth")

		if !ok {
			fmt.Println("You are not logged in. Login with `cmd login` first before setting a webhook secret")
			os.Exit(1)
		}

		var auth types.TestAuth

		err := json.Unmarshal([]byte(cfg), &auth)

		if err != nil {
			fmt.Println("Error loading config:", err)
			os.Exit(1)
		}

		var secret string

		suggestedSecret := crypto.RandString(48)

		fmt.Println(helpers.BotSettingsHelp())

		fmt.Println("Here's a possible good/strong secret:", suggestedSecret, "\nyou can use this for 'Webhook Secret' or generate your own")

		for len(secret) > 0 {
			secret = helpers.GetPassword("Please enter the webhook secret you set on your bot's settings page")
		}

		// Write the config
		err = helpers.WriteConfig("secret", types.WebhookSecret{
			Secret: secret,
		})

		if err != nil {
			fmt.Println("Error writing config:", err)
			os.Exit(1)
		}
	},
}

func init() {
	// login
	cfgCmd.AddCommand(setWebhookSecretCmd)
	cfgCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(cfgCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// cfgCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// cfgCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
