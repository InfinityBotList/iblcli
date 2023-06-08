/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"bytes"
	"crypto/sha512"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/InfinityBotList/ibl/internal/downloader"
	"github.com/InfinityBotList/ibl/internal/links"
	"github.com/fynelabs/selfupdate"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "IBLCLI update command",
	Long:    `IBLCLI updater.`,
	Aliases: []string{"u", "upd"},
	Run: func(cmd *cobra.Command, args []string) {
		force := cmd.Flag("force").Value.String()

		// Check if an update is even required
		updCheckUrl := links.GetCdnURL() + "/dev/" + ProjectName + "/current_rev"
		fmt.Println("Checking for updates (url: " + updCheckUrl)
		currRev, err := downloader.DownloadFileWithProgress(updCheckUrl)

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

		fmt.Println("Updating to version", string(currRev), ", current version:", BuildRev)
		fmt.Print("Continue? [y/N]: ")

		var input string

		fmt.Scanln(&input)

		if strings.ToLower(input) != "y" && strings.ToLower(input) != "yes" {
			fmt.Println("Update cancelled")
			return
		}

		var iblFile = "ibl"

		binFileName := iblFile

		if runtime.GOOS == "windows" {
			binFileName = "ibl.exe"
		}

		url := links.GetCdnURL() + "/dev/" + ProjectName + "/" + runtime.GOOS + "/" + runtime.GOARCH + "/" + binFileName

		fmt.Println("Downloading latest version from:", url)

		execBytes, err := downloader.DownloadFileWithProgress(url)

		if err != nil {
			fmt.Println("Error downloading file:", err)
			return
		}

		if os.Getenv("NO_SHASUM") != "true" {
			shasum := links.GetCdnURL() + "/dev/" + ProjectName + "/" + runtime.GOOS + "/" + runtime.GOARCH + "/" + iblFile + ".sha512"

			fmt.Println("Downloading shasum from:", shasum)

			shasumBytes, err := downloader.DownloadFileWithProgress(shasum)

			if err != nil {
				fmt.Println("Error downloading shasum:", err)
				return
			}

			shasumStr := strings.ReplaceAll(string(shasumBytes), "\n", "")

			fmt.Println("")

			fmt.Println("Verifying shasum...")

			// Create sha512 of the downloaded file
			h := fmt.Sprintf("%x", sha512.Sum512(execBytes)) + "  bin/" + runtime.GOOS + "/" + runtime.GOARCH + "/" + iblFile

			fmt.Println("Expected:", shasumStr)
			fmt.Println("Got:", h)
			fmt.Println("FSize:", "shaSum =", len(shasumStr), "h =", len(h))
			fmt.Println("Match:", h == shasumStr)

			if h != shasumStr {
				fmt.Println("Shasum verification failed!")
				return
			}

			fmt.Println("Shasum verification successful!")
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
