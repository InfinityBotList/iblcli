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
	Use:   "build",
	Short: "Build an IBL service",
	Long:  `Builds an IBL service`,
	Run: func(cmd *cobra.Command, args []string) {
		// Open pkg.yaml
		fmt.Print(ui.BoldText("[INIT] Opening pkg.yaml"))

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

		buildpkg.Build(pkg, "build")
	},
}

// buildPkgCommand allows for the building of packages
var deployPkgCommand = &cobra.Command{
	Use:   "deploy",
	Short: "Deploys an update to an IBL service",
	Long:  `Deploys an update to an IBL service`,
	Run: func(cmd *cobra.Command, args []string) {
		// Open pkg.yaml
		fmt.Print(ui.BoldText("[INIT] Opening pkg.yaml"))

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

		buildpkg.Build(pkg, "deploy")
	},
}

// pkgCmd represents the pkg command
var pkgCmd = &cobra.Command{
	Use:   "pkg",
	Short: "Package (building etc) operations",
	Long:  `Package (building etc) operations`,
}

func init() {
	if devmode.DevMode().Allows(types.DevModeLocal) {
		pkgCmd.AddCommand(buildPkgCommand)
		pkgCmd.AddCommand(deployPkgCommand)
		rootCmd.AddCommand(pkgCmd)
	}
}
