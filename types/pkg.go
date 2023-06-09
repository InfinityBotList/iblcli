package types

// BuildPackage represents the format of a ibl pkg
type BuildPackage struct {
	Language   string                  `yaml:"language" validate:"required"`        // Language of the package
	LangOpts   map[string]string       `yaml:"lang_opts"`                           // Language specific options
	Submodules []BuildPackageSubmodule `yaml:"submodules" validate:"dive,required"` // List of submodules
	Env        map[string]string       `yaml:"env"`                                 // The default env to use                           // Environment variables

	// Common flags
	Project string `yaml:"project" validate:"required"` // Name of the project
	Binary  string `yaml:"binary" validate:"required"`  // Name of the binary
	Service string `yaml:"service"`                     // Name of the service
}

type BuildPackageSubmodule struct {
	Name string `yaml:"name" validate:"required"` // Name of the submodule
	Path string `yaml:"path" validate:"required"` // Path to the submodule
}
