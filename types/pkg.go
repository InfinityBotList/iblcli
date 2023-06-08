package types

// BuildPackage represents the format of a ibl pkg
type BuildPackage struct {
	Rust *BuildPackageRust `yaml:"rust" validate:"required,oneof=rust"`
}

type BuildPackageRust struct {
	Project  string `yaml:"project" validate:"required"` // Name of the project
	Binary   string `yaml:"binary" validate:"required"`  // Name of the binary
	Service  string `yaml:"service"`                     // Name of the service
	Bindings string `yaml:"bindings"`                    // Name of the bindings
}
