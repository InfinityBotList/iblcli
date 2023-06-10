/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/InfinityBotList/ibl/internal/agents/funneleditor"
	"github.com/InfinityBotList/ibl/internal/config"
	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/internal/views"
	"github.com/InfinityBotList/ibl/types"
	"github.com/spf13/cobra"
)

// webserverCmd represents the webserver command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Sets up the a bot for webhooks.",
	Long:  "Sets up a bot for webhooks.",
	Run: func(cmd *cobra.Command, args []string) {
		auth, err := views.AccountSwitcher(string(types.TargetTypeUser))

		if err != nil {
			fmt.Print(ui.RedText("Error getting account info: " + err.Error() + ", exiting..."))
			os.Exit(1)
		}

		if os.Getenv("DEBUG") == "true" {
			fmt.Println("AuthSwitcher:", auth) // temporary to avoid a compile error
		}

		var funnels *types.FunnelList
		err = config.LoadConfig("funnels", &funnels)

		if err != nil {
			fmt.Print(ui.RedText("No valid funnel config found, resetting"))
			funnels = &types.FunnelList{}
		}

		funneleditor.ManageConsole(*auth, *funnels)
	},
}

// adminCmd represents the admin command
var webhCmd = &cobra.Command{
	Use:     "webhook",
	Short:   "Webhook setup and funneling",
	Long:    `Webhook setup and funneling`,
	Aliases: []string{"webhooks", "funnel", "funnels"},
}

func init() {
	webhCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(webhCmd)
}
