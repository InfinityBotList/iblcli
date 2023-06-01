/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"os/exec"

	"github.com/InfinityBotList/ibl/types"
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
	if DevMode == types.DevModeFull {
		rootCmd.AddCommand(logCmd)
		logCmd.Flags().StringP("lines", "l", "300", "Number of lines to show")
		logCmd.Flags().BoolP("begin", "b", false, "Start at beginning of log")
	}
}
