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

		if runtime.GOOS == "windows" {
			url += ".exe"
		}

		fmt.Println("Downloading latest version from:", url)

		bytes, err := helpers.DownloadFileWithProgress(url)

		if err != nil {
			fmt.Println("Error downloading file:", err)
			return
		}

		f, err := os.Create(binFileName + ".new")

		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}

		defer f.Close()

		_, err = f.Write(bytes)

		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}

		// Set file permissions to executable
		err = os.Chmod(binFileName+".new", 0755)

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

		// Set env var to 1
		os.Setenv("IN_UPDATE", "1")

		// Spawn new process
		proc, err := os.StartProcess(binFileName+".new", os.Args, &os.ProcAttr{
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		})

		if err != nil {
			fmt.Println("Error spawning new process:", err)
			return
		}

		// Wait for the new process to exit
		_, err = proc.Wait()

		if err != nil {
			fmt.Println("Error waiting for new process:", err)
			return
		}
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
