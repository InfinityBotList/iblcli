/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates IBL CLI",
	Long:  `Updates IBL CLI to the latest version.`,
	Run: func(cmd *cobra.Command, args []string) {
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

		pcPath := binFileName

		// Check for ~/go/bin
		_, err = os.Stat(os.Getenv("HOME") + "/go/bin/" + binFileName)

		if err == nil {
			pcPath = os.Getenv("HOME") + "/go/bin/" + binFileName
		}

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

		fmt.Println("Created new binary, now replacing old one")

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

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// updateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
