package helpers

import "github.com/InfinityBotList/ibl/types"

var dmCache *types.DevMode

func DevMode() types.DevMode {
	if dmCache != nil {
		return *dmCache
	}

	// Look for devmode config
	var DevMode types.DevMode
	var mode types.DevModeCfg
	err := LoadAndMarshalConfig("dev", &mode)

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

	return DevMode
}
