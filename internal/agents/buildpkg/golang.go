package buildpkg

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/InfinityBotList/ibl/internal/links"
	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/types"
	"github.com/pkg/errors"
)

var golang = map[string][]action{
	"build": {
		{
			Name: "Check compilers",
			Func: func(cfg types.BuildPackage) error {
				defer func() {
					err := recover()

					if err != nil {
						fmt.Print(ui.RedText(" FAILED: " + err.(error).Error()))
						os.Exit(1)
					}
				}()

				// Check go version
				// Ensure these commands exist
				fmt.Print("=> ", "go")

				cmdExec := exec.Command("go", "version")

				outp, err := cmdExec.Output()

				if err != nil {
					fmt.Print(ui.RedText(" NOT FOUND:  " + err.Error() + "]"))
					return errors.New("command not found: go")
				}

				// Get go version
				goVersion := strings.Split(string(outp), " ")[2]

				// Check if go version is >= 1.20.3
				if goVersion < "1.20.3" {
					fmt.Print(ui.RedText(" version < 1.20.3"))
					return errors.New("go version must be >= 1.20.3")
				}

				fmt.Print(ui.GreenText(" OK (" + cmdExec.Path + " " + goVersion + ")"))

				return nil
			},
		},
		{
			Name: "Building package",
			Func: func(cfg types.BuildPackage) error {
				env := map[string]string{
					"CGO_ENABLED": "0",
				}

				_, err := setEnv(env)

				if err != nil {
					return errors.Wrap(err, "Failed to set environment variables")
				}

				// Build package
				fmt.Print("=> (build) go build")

				cmdExec := exec.Command("go", "build", "-v", "-o", cfg.Binary)

				cmdExec.Stdout = os.Stdout
				cmdExec.Stderr = os.Stderr

				if err := cmdExec.Run(); err != nil {
					fmt.Print(ui.RedText(" FAILED: " + err.Error()))
					return errors.Wrap(err, "Failed to build package")
				}

				fmt.Print(ui.GreenText(" OK"))

				return nil
			},
		},
	},
	"deploy": {
		// stop the service
		commonStopExistingService,

		// Rust specific, location of binary differs between languages
		{
			Name: "Copy binary to server",
			Func: func(cfg types.BuildPackage) error {
				sshCmd := exec.Command("scp", cfg.Binary, links.GetVpsSSH()+":~/"+cfg.Project+"/"+cfg.Binary)

				sshCmd.Stdout = os.Stdout
				sshCmd.Stderr = os.Stderr

				if err := sshCmd.Run(); err != nil {
					return errors.Wrap(err, "failed to copy binary to server")
				}

				fmt.Print(ui.GreenText("Successfully copied binary to server"))

				return nil
			},
		},
		commonStartService,
	},
}
