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

// TODO
const CrossCompileNote = `

-- MacOS cross compile --

1. Follow https://stackoverflow.com/questions/40424255/cross-compilation-to-x86-64-unknown-linux-gnu-fails-on-mac-osx
2. Use https://github.com/MaterializeInc/homebrew-crosstools for cross compiling as it is newer
3. Update path: "PATH=/opt/homebrew/Cellar/x86_64-unknown-linux-gnu/0.1.0/bin:$PATH"

## Not always needed, try running make cross before doing the below

4. Symlink gcc if needed by ring at /opt/homebrew/Cellar/x86_64-unknown-linux-gnu/0.1.0/bin based on the error you get
5. Replace 0.1.0 with whatever gcc version you need
6. If you face any build issues on macOS, try removing /opt/homebrew/bin/x86_64-linux-gnu-gcc and then ln -sf /opt/homebrew/bin/x86_64-unknown-linux-gnu-cc /opt/homebrew/bin/x86_64-linux-gnu-gcc
`

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
