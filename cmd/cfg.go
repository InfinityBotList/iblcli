/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/InfinityBotList/ibl/types"
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
		fmt.Print("Auth Type (bot/user/server): ")

		var authType string

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

		fmt.Print("Target ID (user/bot/server ID): ")

		var targetID string

		fmt.Scanln(&targetID)

		fmt.Print("API Token [you can get this from bot/profile/server settings]: ")

		var token string

		fmt.Scanln(&token)

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

		cfgFile := helpers.ConfigFile()

		// Write the config
		err = helpers.Write(cfgFile+"/auth", types.TestAuth{
			AuthType: targetType,
			TargetID: targetID,
			Token:    token,
		})

		if err != nil {
			fmt.Println("Error writing config:", err)
			os.Exit(1)
		}
	},
}

func init() {
	// login
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
