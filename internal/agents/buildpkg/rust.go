package buildpkg

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/InfinityBotList/ibl/internal/links"
	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/types"
)

var rust = map[string][]action{
	"build": {
		{
			Name: "Cleanup",
			Func: func(cfg types.BuildPackage) error {
				// Remove .generated folder
				fmt.Print("=> (cleanup) .generated:")

				err := os.RemoveAll(".generated")

				if err != nil {
					fmt.Print(ui.RedText(" FAILED: " + err.Error()))
					return errors.Wrap(err, "Failed to remove .generated folder")
				}

				fmt.Print(ui.GreenText(" REMOVED"))

				return nil
			},
		},
		{
			Name: "Check compilers",
			Func: func(cfg types.BuildPackage) error {
				if runtime.GOARCH == "amd64" {
					// Assume correct compiler, not cross compiling release
					fmt.Print(ui.YellowText("Not cross compiling release, skipping compiler check"))
					return nil
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
						fmt.Print(ui.RedText(" NOT FOUND:  " + err.Error() + "]"))
						return errors.New("command not found: " + cmd)
					}

					fmt.Print(ui.GreenText(" OK (" + cmdExec.Path + ")"))
				}

				return nil
			},
		},
		{
			Name: "Setup environment",
			Func: func(cfg types.BuildPackage) error {
				if len(cfg.Env) == 0 {
					fmt.Print(ui.YellowText("No environment variables to set, skipping"))
					return nil
				}

				envFile, err := setEnv(cfg.Env)

				if err != nil {
					return err
				}

				// Create .env file if it doesn't exist
				f, err := os.Create(".env")

				if err != nil {
					return errors.Wrap(err, "failed to create .env file")
				}

				defer f.Close()

				// Write envfile to .env
				_, err = f.WriteString(strings.Join(envFile, "\n"))

				if err != nil {
					return errors.Wrap(err, "failed to write envfile to .env file")
				}

				return nil
			},
		},
		{
			Name: "Building package",
			Func: func(cfg types.BuildPackage) error {
				env := map[string]string{
					"RUST_BACKTRACE":   "1",
					"CARGO_TERM_COLOR": "always",
				}

				var args []string

				if runtime.GOOS != "amd64" {
					target := getTarget()
					// Set cross compiler env
					env["CARGO_TARGET_GNU_LINKER"] = target + "-gcc"
					env["CARGO_TARGET_X86_64_UNKNOWN_LINUX_GNU_LINKER"] = target + "-gcc"

					args = []string{
						"cargo",
						"build",
						"--target=" + target,
						"--release",
					}
				} else {
					// Assume correct compiler, not cross compiling release
					fmt.Print(ui.YellowText("Not cross compiling release, not setting cross compile env"))
					if os.Getenv("NO_SET_RUSTFLAGS") == "" {
						rfLocal := `-C target-cpu=native -C link-arg=-fuse-ld=lld`

						if os.Getenv("RUSTFLAGS") != "" {
							rfLocal += " " + os.Getenv("RUSTFLAGS")
						}

						env["RUSTFLAGS"] = rfLocal
					}

					args = []string{
						"cargo",
						"build",
						"--release",
					}
				}

				// Set env
				_, err := setEnv(env)

				if err != nil {
					return err
				}

				cmd := exec.Command(args[0], args[1:]...)

				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Env = os.Environ()

				if err := cmd.Run(); err != nil {
					return errors.Wrap(err, "failed to build package")
				}

				fmt.Print(ui.GreenText("Successfully built package"))

				return nil
			},
		},
		{
			Name: "Building bindings",
			Func: func(cfg types.BuildPackage) error {
				bindingsLoc, ok := cfg.LangOpts["bindings"]

				if !ok || bindingsLoc == "" {
					fmt.Print(ui.GreenText("No bindings to build"))
					return nil
				}

				// Here, we use cargo test to call ts-rs binding generation
				env := map[string]string{
					"RUST_BACKTRACE":   "1",
					"CARGO_TERM_COLOR": "always",
				}

				// Set env
				_, err := setEnv(env)

				if err != nil {
					return err
				}

				cmd := exec.Command("cargo", "test")

				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				if err := cmd.Run(); err != nil {
					return errors.Wrap(err, "failed to build bindings")
				}

				fmt.Print(ui.GreenText("Successfully built bindings"))

				return nil
			},
		},
	},
	"deploy": {
		{
			Name: "Stop existing service",
			Func: func(cfg types.BuildPackage) error {
				sshCmd := exec.Command("ssh", links.GetVpsSSH(), "sudo", "systemctl", "stop", cfg.Service)

				sshCmd.Stdout = os.Stdout
				sshCmd.Stderr = os.Stderr

				if err := sshCmd.Run(); err != nil {
					return errors.Wrap(err, "failed to stop service")
				}

				fmt.Print(ui.GreenText("Successfully stopped service"))

				return nil
			},
		},
		{
			Name: "Copy binary to server",
			Func: func(cfg types.BuildPackage) error {
				target := getTarget()
				sshCmd := exec.Command("scp", "target/"+target+"/release/"+cfg.Binary, links.GetVpsSSH()+":~/"+cfg.Project+"/"+cfg.Binary)

				sshCmd.Stdout = os.Stdout
				sshCmd.Stderr = os.Stderr

				if err := sshCmd.Run(); err != nil {
					return errors.Wrap(err, "failed to copy binary to server")
				}

				fmt.Print(ui.GreenText("Successfully copied binary to server"))

				return nil
			},
		},
		{
			Name: "Start service",
			Func: func(cfg types.BuildPackage) error {
				sshCmd := exec.Command("ssh", links.GetVpsSSH(), "sudo", "systemctl", "start", cfg.Service)

				sshCmd.Stdout = os.Stdout
				sshCmd.Stderr = os.Stderr

				if err := sshCmd.Run(); err != nil {
					return errors.Wrap(err, "failed to start service")
				}

				fmt.Print(ui.GreenText("Successfully started service"))

				return nil
			},
		},
		{
			Name: "Copy bindings",
			Func: func(cfg types.BuildPackage) error {
				bindingsLoc, ok := cfg.LangOpts["bindings"]

				if !ok || bindingsLoc == "" {
					fmt.Print(ui.GreenText("No bindings to copy"))
					return nil
				}

				sshCmd := exec.Command("scp", "-r", ".generated", links.GetVpsSSH()+":"+bindingsLoc)

				sshCmd.Stdout = os.Stdout
				sshCmd.Stderr = os.Stderr

				if err := sshCmd.Run(); err != nil {
					return errors.Wrap(err, "failed to copy bindings to server")
				}

				fmt.Print(ui.GreenText("Successfully copied bindings to server"))

				fmt.Print(ui.BoldText("Moving bindings to correct location"))

				cmd := []string{
					"ssh",
					links.GetVpsSSH(),
					"rm",
					"-rf",
					bindingsLoc + "/*.ts",
					"&&",
					"cp",
					"-rf",
					bindingsLoc + "/.generated/*.ts",
					bindingsLoc + "/",
					"&&",
					"rm",
					"-rf",
					bindingsLoc + "/.generated",
				}

				fmt.Print(ui.BoldText("Running command: " + strings.Join(cmd, " ")))

				sshCmd = exec.Command(cmd[0], cmd[1:]...)

				sshCmd.Stdout = os.Stdout
				sshCmd.Stderr = os.Stderr

				if err := sshCmd.Run(); err != nil {
					return errors.Wrap(err, "failed to move bindings to correct location")
				}

				fmt.Print(ui.GreenText("Successfully moved bindings to correct location"))

				return nil
			},
		},
	},
}
