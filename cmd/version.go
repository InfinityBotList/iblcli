/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"fmt"
	"runtime"

	"github.com/InfinityBotList/ibl/internal/devmode"
	"github.com/InfinityBotList/ibl/internal/iblfile"
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
		fmt.Println("iblFileProtocol:", iblfile.Protocol)
		fmt.Println("goInfo:", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		fmt.Println("devMode:", devmode.DevMode())

		fmt.Println("\nCopyright © 2022 Infinity Bot List")
		fmt.Println("Licensed under the MIT license. See LICENSE file in the project root for full license information.")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
