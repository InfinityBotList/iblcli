/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/InfinityBotList/ibl/internal/lib"
	"github.com/InfinityBotList/ibl/types"
	"github.com/spf13/cobra"
)

// webserverCmd represents the webserver command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Sets up the a bot for webhooks.",
	Long:  "Sets up a bot for webhooks.",
	Run: func(cmd *cobra.Command, args []string) {
		auth := lib.AccountSwitcher()

		fmt.Println("AuthSwitcher:", auth) // temporary to avoid a compile error

		var funnels *types.FunnelList
		err := helpers.LoadAndMarshalConfig("funnels", &funnels)

		if err != nil {
			port := helpers.GetInput("What port should the webserver run on?", func(s string) bool {
				// Check if port is a number
				_, err := strconv.Atoi(s)

				if err != nil {
					fmt.Fprint(os.Stderr, helpers.RedText("Invalid port number"))
					return false
				}

				return true
			})

			// Write funnels file
			portNum, err := strconv.Atoi(port)

			if err != nil {
				fmt.Fprint(os.Stderr, helpers.RedText("Invalid port number"))
				os.Exit(1)
			}

			funnels = &types.FunnelList{
				Port:    portNum,
				Funnels: []types.WebhookFunnel{},
			}

			err = helpers.WriteConfig("funnels", funnels)

			if err != nil {
				fmt.Fprint(os.Stderr, helpers.RedText("Config save error: "+err.Error()))
				os.Exit(1)
			}
		}
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
