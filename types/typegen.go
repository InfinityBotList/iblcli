package types

// TypeGen represents the format of a ibl typegen config
type TypeGen struct {
	Path     string   `yaml:"path" validate:"required"`     // Path to copy types from
	Projects []string `yaml:"projects" validate:"required"` // List of projects to copy types to
}
