package types

// BuildPackage represents the format of a ibl pkg
type BuildPackage struct {
	Lang map[string]BuildPackageInner `yaml:",inline" validate:"required"`
}

type BuildPackageInner struct {
	Project  string `yaml:"project" validate:"required"` // Name of the project
	Binary   string `yaml:"binary" validate:"required"`  // Name of the binary
	Service  string `yaml:"service"`                     // Name of the service
	Bindings string `yaml:"bindings"`                    // Name of the bindings
}
