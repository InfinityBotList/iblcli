package buildpkg

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/types"
)

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

const DefaultTarget = "x86_64-unknown-linux-gnu"

// Internal struct
type action struct {
	Name string
	Func func(cfg types.BuildPackage)
}

var RustBuildActions = []action{
	{
		Name: "Check compilers",
		Func: func(cfg types.BuildPackage) {
			if runtime.GOARCH == "amd64" {
				// Assume correct compiler, not cross compiling release
				fmt.Println(ui.YellowText("Not cross compiling release, skipping compiler check"))
				return
			}

			for _, cmd := range []string{
				"rustc",
				"cargo",
				"x86_64-linux-gnu-gcc",
				"x86_64-unknown-linux-gnu-gcc",
			} {
				// Ensure these commands exist
				fmt.Print("=> ", cmd)

				cmdExec := exec.Command(cmd, "--version")

				if _, err := cmdExec.Output(); err != nil {
					fmt.Println(ui.RedText(" NOT FOUND:  " + err.Error() + "]"))
				}

				fmt.Print(ui.GreenText(" OK (" + cmdExec.Path + ")"))
			}
		},
	},
	{
		Name: "Set correct database url",
		Func: func(cfg types.BuildPackage) {
			if os.Getenv("DATABASE_URL") == "" {
				fmt.Println(ui.YellowText("DATABASE_URL not set, setting to default: postgres://localhost/infinity"))
				os.Setenv("DATABASE_URL", "postgres://localhost/infinity")
			}

			// Create .env file if it doesn't exist
			f, err := os.Create(".env")

			if err != nil {
				panic(err)
			}

			defer f.Close()

			// Write DATABASE_URL to .env
			_, err = f.WriteString("DATABASE_URL=" + os.Getenv("DATABASE_URL"))

			if err != nil {
				panic(err)
			}
		},
	},
	{
		Name: "Building package",
		Func: func(cfg types.BuildPackage) {
			os.Setenv("CARGO_TERM_COLOR", "always") // Force color output

			var args []string

			if runtime.GOOS != "amd64" {
				var target string

				if os.Getenv("TARGET") != "" {
					target = os.Getenv("TARGET")
				} else {
					target = DefaultTarget
				}

				// Set cross compiler env
				os.Setenv("CARGO_TARGET_GNU_LINKER", target+"-gcc")
				os.Setenv("CARGO_TARGET_X86_64_UNKNOWN_LINUX_GNU_LINKER", target+"-gcc")

				args = []string{
					"cargo",
					"build",
					"--target=" + target,
					"--release",
				}
			} else {
				// Assume correct compiler, not cross compiling release
				fmt.Println(ui.YellowText("Not cross compiling release, not setting cross compile env"))
				if os.Getenv("NO_SET_RUSTFLAGS") == "" {
					rfLocal := `-C target-cpu=native -C link-arg=-fuse-ld=lld`

					if os.Getenv("RUSTFLAGS") != "" {
						rfLocal += " " + os.Getenv("RUSTFLAGS")
					}

					os.Setenv("RUSTFLAGS", rfLocal)
				}

				args = []string{
					"cargo",
					"build",
					"--release",
				}
			}

			cmd := exec.Command(args[0], args[1:]...)

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Env = os.Environ()

			if err := cmd.Run(); err != nil {
				fmt.Println(ui.RedText("Failed to build package: " + err.Error()))
				os.Exit(1)
			}

			fmt.Println(ui.GreenText("Successfully built package"))
		},
	},
}

func Build(cfg types.BuildPackage) {
	var actions []action
	if cfg.Rust != nil {
		fmt.Println(ui.BoldText("[INIT] Using rust build system"))
		actions = RustBuildActions
	}

	for i, a := range actions {
		fmt.Print(ui.BoldText(fmt.Sprintf("[%d/%d] %s\n", i+1, len(actions), a.Name)))
		a.Func(cfg)
	}
}
