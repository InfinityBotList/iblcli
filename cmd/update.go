/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/fynelabs/selfupdate"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Updates IBL CLI",
	Long:    `Updates IBL CLI to the latest version.`,
	Aliases: []string{"u", "upd"},
	Run: func(cmd *cobra.Command, args []string) {
		force := cmd.Flag("force").Value.String()

		// Check if an update is even required
		updCheckUrl := helpers.GetAssetsURL() + "/" + ProjectName + "/current_rev"
		fmt.Println("Checking for updates (url: " + updCheckUrl)
		currRev, err := helpers.DownloadFileWithProgress(updCheckUrl)

		if err != nil {
			fmt.Println("Error checking for updates:", err)
			return
		}

		if string(currRev) == BuildRev {
			fmt.Println("You are already on the latest version!")

			if force == "true" {
				fmt.Println("Force flag set, continuing")
			} else {
				return
			}
		}

		fmt.Println("Updating to version", string(currRev))
		fmt.Print("Continue? [y/N]: ")

		var input string

		fmt.Scanln(&input)

		if strings.ToLower(input) != "y" && strings.ToLower(input) != "yes" {
			fmt.Println("Update cancelled")
			return
		}

		binFileName := "ibl"

		if runtime.GOOS == "windows" {
			binFileName = "ibl.exe"
		}

		url := helpers.GetAssetsURL() + "/" + ProjectName + "/" + runtime.GOOS + "/" + runtime.GOARCH + "/" + binFileName

		fmt.Println("Downloading latest version from:", url)

		execBytes, err := helpers.DownloadFileWithProgress(url)

		if err != nil {
			fmt.Println("Error downloading file:", err)
			return
		}

		err = selfupdate.Apply(bytes.NewReader(execBytes), selfupdate.Options{})
		if err != nil {
			// error handling
			fmt.Println("Error updating:", err)
		}

		fmt.Println("Updated successfully!")
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// updateCmd.PersistentFlags().String("foo", "", "A help for foo")
	updateCmd.Flags().BoolP("force", "f", false, "Force a update/redownload even if the latest version is already installed")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// updateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
