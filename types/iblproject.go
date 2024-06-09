package types

// IBLProject represents the format of a ibl project.yaml file
type IBLProject struct {
	TypeGen *TypeGen `yaml:"typegen"` // `ibl typegen` config
}
