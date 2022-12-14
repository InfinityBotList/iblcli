/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Version Information",
	Long:    `Version Information`,
	Aliases: []string{"v"},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("version:", GitCommit)
		fmt.Println("buildRev:", BuildRev)
		fmt.Println("buildTime:", BuildTime)
		fmt.Println("seedApiVersion:", seedApiVer)

		fmt.Println("\nCopyright © 2022 Infinity Bot List")
		fmt.Println("Licensed under the MIT license. See LICENSE file in the project root for full license information.")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
