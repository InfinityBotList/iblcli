/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/InfinityBotList/ibl/internal/config"
	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/internal/views"
	"github.com/InfinityBotList/ibl/types"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:     "login TYPE",
	Short:   "Login to the IBL API",
	Long:    `Login to the IBL API using a bot/user/server token.`,
	Aliases: []string{"auth", "a", "l"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, err := views.AccountSwitcher(args[0])

		if err != nil {
			fmt.Print(ui.RedText("Error logging in: " + err.Error()))
			os.Exit(1)
		}
	},
}

var devModeToggle = &cobra.Command{
	Use:   "toggledev",
	Short: "Toggle dev mode",
	Long:  "off = disable dev mode\nlocal = locally performable actions\nfull = all commands",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get dev mode flag
		devMode := args[0]

		if devMode != "off" && devMode != "full" && devMode != "local" {
			panic("Invalid dev mode")
		}

		var DevMode types.DevMode

		switch devMode {
		case "off":
			DevMode = types.DevModeOff
		case "full":
			DevMode = types.DevModeFull
		case "local":
			DevMode = types.DevModeLocal
		}

		fmt.Print(ui.YellowText("WARNING: Developer mode is enabled, use at your own risk"))

		// Write dev mode to config
		config.WriteConfig("dev", types.DevModeCfg{
			Mode: DevMode,
		})
	},
}

func init() {
	// login
	rootCmd.AddCommand(devModeToggle)
	rootCmd.AddCommand(loginCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// cfgCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// cfgCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
