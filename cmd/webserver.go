/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/spf13/cobra"
)

// webserverCmd represents the webserver command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Sets up the webserver for webhooks/vote rewards to allow you to easily handle voting.",
	Long:  "Sets up the webserver for webhooks/vote rewards to allow you to easily handle voting.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(helpers.BoldBlueText(helpers.AddUnderDecor("Prep Work [1/2]")))
		var flag bool = true
		for flag {
			_, ok := helpers.LoadConfig("auth")

			if !ok {
				fmt.Println("You are not logged in on IBL CLI yet! Performing login now!")
				loginCmd.Run(cmd, args)
			} else {
				fmt.Println("Excellent! You're logged in!")
				flag = false
			}
		}

		flag = true
		for flag {
			_, ok := helpers.LoadConfig("secret")

			if !ok {
				fmt.Print(helpers.RedText("You don't have a webhook secret set yet!"))
				fmt.Print(helpers.RedText("For security purposes, it is HIGHLY recommended that you set a webhook secret. Otherwise, the API Token will be used"))

				setupQa := helpers.GetInput("Do you want to set a webhook secret? (y/n)", func(s string) bool {
					if s == "y" || s == "n" {
						return true
					} else {
						fmt.Println("Invalid response. Please enter y or n")
						return false
					}
				})

				if setupQa == "y" {
					setWebhookSecretCmd.Run(cmd, args)
				} else {
					fmt.Println("Skipping webhook secret setup...")
					flag = false
				}
			} else {
				fmt.Println("Excellent! Your webhook secret appears to be set!")
				flag = false
			}
		}

		// Choose notification method
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// webserverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// webserverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
