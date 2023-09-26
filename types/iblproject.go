package types

// IBLProject represents the format of a ibl project.yaml file
type IBLProject struct {
	Pkg     *BuildPackage `yaml:"pkg" validate:"dive"`     // `ibl pkg` config
	TypeGen *TypeGen      `yaml:"typegen" validate:"dive"` // `ibl typegen` config
}
