/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

// Store git commit hash
var GitCommit string

func init() {
	// Use runtime to get the git commit hash
	GitCommit = runtime.Version()
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ibl",
	Short: "IBL is a simple utility to make development of IBL easier as well as to allow bot developers to test their bots.",
	Long: `IBL is a simple utility to make development of Infinity Bot List easier as well as to allow bot developers to test the API. 

For more information, try running "ibl --help"

If you wish to add a new command, use "~/go/bin/cobra-cli add"`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	if os.Getenv("IN_UPDATE") == "1" {
		fmt.Println("Update successful, now on version:", GitCommit)

		// Try to kill the old process

		proc, err := os.FindProcess(os.Getppid())

		if err != nil {
			fmt.Println("Error finding parent process:", err)
		}

		err = proc.Kill()

		if err != nil {
			fmt.Println("Error killing parent process:", err)
		}

		// Delete old binary
		err = os.Remove("ibl")
		if err != nil {
			fmt.Println("Error renaming file:", err)
			return
		}
		// Rename new binary
		err = os.Rename("ibl.new", "ibl")
		if err != nil {
			fmt.Println("Error renaming file:", err)
			return
		}
		// Exit
		os.Exit(0)
	}

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ibl.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
