// Package devmode defines the core runtime of iblcli
package devmode

import (
	"github.com/InfinityBotList/ibl/internal/config"
	"github.com/InfinityBotList/ibl/types"
)

var dmCache *types.DevMode

// Returns the dev mode status of the CLI
func DevMode() types.DevMode {
	if dmCache != nil {
		return *dmCache
	}

	// Look for devmode config
	var DevMode types.DevMode
	var mode types.DevModeCfg
	err := config.LoadConfig("dev", &mode)

	if err != nil {
		DevMode = types.DevModeOff
	} else {
		switch mode.Mode {
		case "off":
			DevMode = types.DevModeOff
		case "local":
			DevMode = types.DevModeLocal
		case "full":
			DevMode = types.DevModeFull
		}
	}

	dmCache = &DevMode

	return DevMode
}
