/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"os/exec"

	"github.com/spf13/cobra"
)

// logCmd represents the log command
var logCmd = &cobra.Command{
	Use:     "log SERVICE",
	Short:   "View the logs of a service",
	Long:    `Shows the API logs of Infinity Bot List. You must be logged into the VPS to do this. FOR INTERNAL USE ONLY`,
	Aliases: []string{"logs"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// journalctl -u $1 -n 300 -f
		lines := cmd.Flag("lines").Value.String()
		begin := cmd.Flag("begin").Value.String()

		var journalCmd *exec.Cmd

		cmdArgs := []string{
			"-u",
			args[0],
			"-n",
			lines,
		}

		if begin != "true" {
			cmdArgs = append(cmdArgs, "-f")
		}

		journalCmd = exec.Command("journalctl", cmdArgs...)

		journalCmd.Stdout = cmd.OutOrStdout()
		journalCmd.Stderr = cmd.OutOrStderr()
		journalCmd.Stdin = cmd.InOrStdin()

		journalCmd.Run()
	},
}

func init() {
	rootCmd.AddCommand(logCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// logsCmd.PersistentFlags().String("foo", "", "A help for foo")
	logCmd.Flags().StringP("lines", "l", "300", "Number of lines to show")
	logCmd.Flags().BoolP("begin", "b", false, "Start at beginning of log")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// logsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
