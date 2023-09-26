package types

// IBLProject represents the format of a ibl project.yaml file
type IBLProject struct {
	Pkg     *BuildPackage `yaml:"pkg"`     // `ibl pkg` config
	TypeGen *TypeGen      `yaml:"typegen"` // `ibl typegen` config
}
