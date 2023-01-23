/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/InfinityBotList/ibl/types"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/spf13/cobra"
)

// cfgCmd represents the cfg command
var cfgCmd = &cobra.Command{
	Use:   "cfg",
	Short: "Configuration system",
	Long:  `Configure the "ibl" client (authentication and other settings).`,
}

var loginCmd = &cobra.Command{
	Use:     "login",
	Short:   "Login to the IBL API",
	Long:    `Login to the IBL API using a bot or user token.`,
	Aliases: []string{"auth", "a", "l"},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(helpers.BoldBlueText(helpers.AddUnderDecor("Login")))

		var authType = helpers.GetInput("Auth Type (bot/user/server)", func(s string) bool {
			if strings.ToLower(s) == "bot" || strings.ToLower(s) == "user" || strings.ToLower(s) == "server" {
				return true
			} else {
				fmt.Fprintln(os.Stderr, "Invalid auth type. Choose from bot, user or server")
				return false
			}
		})

		fmt.Scanln(&authType)

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

		var targetID = helpers.GetInput("Target ID (user/bot/server ID)", func(s string) bool {
			return len(s) > 0
		})

		token := helpers.GetPassword("API Token [you can get this from bot/profile/server settings, vanities are also supported if applicable]")

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
		err = helpers.WriteConfig("auth", types.TestAuth{
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

		resp, err := helpers.NewReq().Get("bots/" + auth.TargetID + "/webhook-state").Do()

		if err != nil {
			fmt.Println("Error getting webhook state:", err)
			os.Exit(1)
		}

		var webhookState types.WebhookState
		err = resp.JsonOk(&webhookState)

		if err != nil {
			fmt.Println("Error getting webhook state:", err)
			os.Exit(1)
		}

		var secret string

		if webhookState.SecretSet {
			secret = helpers.GetPassword("Please enter the webhook secret you set on your bot's settings page.\n\nIf you don't know it, regenerate it or unset it (leave it blank and save), rerun setup and we'll provide instructions\n\nSecret")
		} else {
			suggestedSecret := crypto.RandString(48)
			fmt.Println("Seems like you haven't set a webhook secret yet.")

			fmt.Println(helpers.BotSettingsHelp())

			fmt.Println("Here's a possible good/strong secret:", suggestedSecret, "\nyou can use this for 'Webhook Secret' or generate your own")

			os.Exit(1)
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
