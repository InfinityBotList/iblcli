package cmd

import (
	"fmt"
	"os"

	"github.com/InfinityBotList/ibl/internal/agents/buildpkg"
	"github.com/InfinityBotList/ibl/internal/devmode"
	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// buildPkgCommand allows for the building of packages
var buildPkgCommand = &cobra.Command{
	Use:     "build",
	Short:   "Build an IBL service",
	Long:    `Builds an IBL service`,
	Args:    cobra.ExactArgs(2),
	Aliases: []string{"addexperiment", "ae"},
	Run: func(cmd *cobra.Command, args []string) {
		// Open pkg.yaml
		fmt.Println(ui.BoldText("[INIT] Opening pkg.yaml"))

		bytes, err := os.ReadFile("pkg.yaml")

		if err != nil {
			panic(err)
		}

		// Parse pkg.yaml
		var pkg types.BuildPackage

		err = yaml.Unmarshal(bytes, &pkg)

		if err != nil {
			panic(err)
		}

		// Check if the pkg is valid
		err = rootValidator.Struct(pkg)

		if err != nil {
			panic(err)
		}

		buildpkg.Build(pkg)
	},
}

// pkgCmd represents the pkg command
var pkgCmd = &cobra.Command{
	Use:   "pkg",
	Short: "Package (building etc) operations",
	Long:  `Page operations`,
}

func init() {
	if devmode.DevMode().Allows(types.DevModeLocal) {
		pkgCmd.AddCommand(buildPkgCommand)
		rootCmd.AddCommand(pkgCmd)
	}
}
