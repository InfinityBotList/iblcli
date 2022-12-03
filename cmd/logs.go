/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/spf13/cobra"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:     "logs",
	Short:   "View the API logs",
	Long:    `Shows the API logs of Infinity Bot List. The remote server must be configured to allow this.`,
	Aliases: []string{"log"},
	Run: func(cmd *cobra.Command, args []string) {
		remote, err := helpers.GetRemote()

		if err != nil {
			fmt.Println("Error getting remote [try adding it using 'ibl remote set']:", err)
			return
		}

		cli, err := remote.Connect()

		if err != nil {
			fmt.Println("Error connecting to remote:", err)
			return
		}

		// Try to get the server info
		sv := cli.ServerVersion()

		fmt.Println("Connected to remote server with ssh version:", string(sv))
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// logsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// logsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
