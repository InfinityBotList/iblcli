/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/InfinityBotList/ibl/internal/funnel"
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

		if os.Getenv("DEBUG") == "true" {
			fmt.Println("AuthSwitcher:", auth) // temporary to avoid a compile error
		}

		var funnels *types.FunnelList
		err := helpers.LoadConfig("funnels", &funnels)

		if err != nil {
			fmt.Print(helpers.RedText("No valid funnel config found, resetting"))
			funnels = &types.FunnelList{}
		}

		funnel.ManageConsole(auth, *funnels)
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
}
