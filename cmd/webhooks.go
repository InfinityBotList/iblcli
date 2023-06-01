/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/InfinityBotList/ibl/types"
	"github.com/spf13/cobra"
)

// webserverCmd represents the webserver command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Sets up the a bot for webhooks.",
	Long:  "Sets up a bot for webhooks.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(helpers.BoldBlueText(helpers.AddUnderDecor("Prep Work [1/2]")))
		var flag bool = true
		for flag {
			conf, ok := helpers.LoadConfig("auth@user")

			if !ok {
				fmt.Print(helpers.RedText("You are not logged in on IBL CLI yet! A login is required for proper configuration and setup..."))
				os.Setenv("REQUIRED_AUTH_METHOD", "user")
				loginCmd.Run(cmd, args)
			} else {
				var a types.TestAuth

				err := json.Unmarshal([]byte(conf), &a)

				if err != nil {
					fmt.Print(helpers.RedText("Error loading config: " + err.Error() + ", reauthenticating..."))
					os.Setenv("REQUIRED_AUTH_METHOD", "user")
					loginCmd.Run(cmd, args)
				}

				username, err := helpers.GetUsername(a.TargetID)

				if err != nil {
					fmt.Print(helpers.RedText("Error getting username: " + err.Error() + ", reauthenticating..."))
					os.Setenv("REQUIRED_AUTH_METHOD", "user")
					loginCmd.Run(cmd, args)
				}

				confirm := helpers.GetInput(fmt.Sprint("You're logged in as", helpers.BoldText(username), "Continue [y/n]"), func(s string) bool {
					return s == "y" || s == "n"
				})

				if confirm == "n" {
					os.Setenv("REQUIRED_AUTH_METHOD", "user")
					loginCmd.Run(cmd, args)
				}
			}

			fmt.Println("Excellent! You're logged in!")
			flag = false
		}

		flag = true
		for flag {
			_, ok := helpers.LoadConfig("secret")

			if !ok {
				fmt.Print(helpers.RedText("You don't have a webhook secret set yet!"))
				fmt.Print(helpers.RedText("For security purposes, this is required before you can start using the webhook server"))

				setWebhookSecretCmd.Run(cmd, args)
			} else {
				fmt.Println("Excellent! Your webhook secret appears to be set!")
				flag = false
			}
		}

		// Choose notification method
	},
}

// adminCmd represents the admin command
var webhCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Webhook setup and funneling",
	Long:  `Webhook setup and funneling`,
}

func init() {
	webhCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(webhCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// webserverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// webserverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
