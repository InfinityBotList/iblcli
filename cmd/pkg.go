package cmd

import (
	"fmt"
	"os"

	"github.com/InfinityBotList/ibl/internal/agents/buildpkg"
	"github.com/InfinityBotList/ibl/internal/devmode"
	"github.com/InfinityBotList/ibl/internal/projectconfig"
	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/types"
	"github.com/spf13/cobra"
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

// pkgCmd represents the pkg command
var pkgCmd = &cobra.Command{
	Use:   "pkg [build|deploy]",
	Short: "Package build system",
	Long:  `Package build system`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		action := args[0]

		proj, err := projectconfig.LoadProjectConfig()

		if err != nil {
			fmt.Print(ui.RedText("Failed to load project config: " + err.Error()))
			os.Exit(1)
		}

		if proj.Pkg == nil {
			fmt.Print(ui.RedText("No pkg config found in project.yaml"))
			os.Exit(1)
		}

		err = buildpkg.Enter(*proj.Pkg, action)

		if err != nil {
			fmt.Print(ui.RedText(err.Error()))
			os.Exit(1)
		}
	},
}

func init() {
	if devmode.DevMode().Allows(types.DevModeLocal) {
		rootCmd.AddCommand(pkgCmd)
	}
}
