// Common actions for all languages
package buildpkg

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/InfinityBotList/ibl/internal/links"
	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/types"
	"github.com/pkg/errors"
)

// Stops the existing service
var commonStopExistingService = action{
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
}

// Starts the server
var commonStartService = action{
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
}
