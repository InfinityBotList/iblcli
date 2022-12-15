/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/InfinityBotList/ibl/helpers"
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
		updCheckUrl := helpers.GetAssetsURL() + "/shadowsight/current_rev"
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

		url := helpers.GetAssetsURL() + "/shadowsight/" + runtime.GOOS + "/" + runtime.GOARCH + "/" + binFileName

		fmt.Println("Downloading latest version from:", url)

		bytes, err := helpers.DownloadFileWithProgress(url)

		if err != nil {
			fmt.Println("Error downloading file:", err)
			return
		}

		pcPath, err := os.Executable()
		if err != nil {
			panic(err)
		}
		fmt.Println("UpdateBinary:", pcPath)

		var path string
		if runtime.GOOS == "windows" {
			path = pcPath + ".new.exe"
		} else {
			path = pcPath + ".new"
		}
		f, err := os.Create(path)

		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}

		_, err = f.Write(bytes)

		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}

		err = f.Close()

		if err != nil {
			fmt.Println("Error closing file:", err)
			return
		}

		// Set file permissions to executable
		err = os.Chmod(path, 0755)

		if err != nil {
			fmt.Println("Error setting file permissions:", err)
			return
		}

		/*
			Spawn new process using os
			This is so we can delete the old binary
			and rename the new one

			This is also so we can exit the program
			and not have to worry about the old binary
			being deleted
		*/

		fmt.Println("Created new binary, now replacing old one, pcPath:", pcPath)

		// Set env var to 1
		env := []string{
			"IN_UPDATE=1",
			"PC_PATH=" + pcPath,
		}

		// Spawn new process
		_, err = os.StartProcess(path, os.Args, &os.ProcAttr{
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
			Env:   env,
		})

		if err != nil {
			fmt.Println("Error spawning new process:", err)
			return
		}

		os.Exit(0)
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
