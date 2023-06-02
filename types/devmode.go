package types

type DevMode string

const (
	DevModeFull  DevMode = "full"
	DevModeLocal DevMode = "local"
	DevModeOff   DevMode = "off"
)

func (d DevMode) Allows(m DevMode) bool {
	if d == DevModeFull {
		return m == DevModeFull || m == DevModeLocal || m == DevModeOff
	}

	if d == DevModeLocal {
		return m == DevModeLocal || m == DevModeOff
	}

	return m == DevModeOff
}

type DevModeCfg struct {
	Mode DevMode `yaml:"mode"`
}
