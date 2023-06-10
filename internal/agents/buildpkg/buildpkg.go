package buildpkg

import (
	"fmt"
	"os"
	"strings"

	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/types"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const DefaultTarget = "x86_64-unknown-linux-gnu"

var rootValidator *validator.Validate

// Internal struct
type action struct {
	Name string
	Func func(cfg types.BuildPackage) error
}

func setEnv(env map[string]string) (envFile []string, err error) {
	for k, v := range env {
		envFile = append(envFile, k+"="+v+"\n")

		fmt.Print("=> (env) ", k, "=", v)

		if os.Getenv(k) != "" {
			fmt.Print(ui.YellowText(" (already set)"))
			continue
		}

		err = os.Setenv(k, v)

		if err != nil {
			return envFile, errors.Wrap(err, "Failed to set env")
		}

		fmt.Print(ui.GreenText(" SET"))
	}

	return envFile, nil
}

func getTarget() string {
	var target string
	if os.Getenv("TARGET") != "" {
		target = os.Getenv("TARGET")
	} else {
		target = DefaultTarget
	}

	return target
}

var Actions = map[string]map[string][]action{
	"rust":  rust,
	"go":    golang,
	"dummy": dummy,
}

func LoadPackage() (*types.BuildPackage, error) {
	// Open pkg.yaml
	fmt.Print(ui.BoldText("[INIT] Opening pkg.yaml"))

	bytes, err := os.ReadFile("pkg.yaml")

	if err != nil {
		return nil, err
	}

	// Parse pkg.yaml
	var pkg types.BuildPackage

	err = yaml.Unmarshal(bytes, &pkg)

	if err != nil {
		return nil, err
	}

	// Check if the pkg is valid
	err = rootValidator.Struct(pkg)

	if err != nil {
		return nil, err
	}

	return &pkg, nil
}

// Enter takes a package and runs a command on it
func Enter(cfg types.BuildPackage, arg string) error {
	cwd, err := os.Getwd()

	if err != nil {
		return errors.Wrap(err, "Failed to get working directory")
	}

	if len(cfg.Submodules) > 0 {
		for _, submodule := range cfg.Submodules {
			fmt.Print(ui.BoldText("[META] Building submodule '" + submodule.Name + "'"))

			// Change directory to submodule
			err := os.Chdir(submodule.Path)

			if err != nil {
				return errors.Wrap(err, "Failed to change directory to submodule")
			}

			// Load submodule
			subPkg, err := LoadPackage()

			if err != nil {
				return errors.Wrap(err, "Failed to load submodule")
			}

			// Run submodule
			err = Enter(*subPkg, arg)

			if err != nil {
				return errors.Wrap(err, "Failed to build submodule")
			}

			// Change directory back to cwd
			err = os.Chdir(cwd)

			if err != nil {
				return errors.Wrap(err, "Failed to change directory to cwd")
			}
		}
	}

	fmt.Print(ui.BoldText("[INIT] Using " + cfg.Language + " build system"))

	langAction, ok := Actions[cfg.Language]

	if !ok {
		return errors.New("Unsupported language: " + cfg.Language)
	}

	actions, ok := langAction[arg]

	if !ok {
		return errors.New("Unsupported action: " + arg)
	}

	if len(actions) == 0 {
		return errors.New("No actions found for " + arg)
	}

	if len(cfg.Env) > 0 {
		// Setup env, this is a core task
		fmt.Print(ui.BoldText("[CORE] Setting up environment"))

		envFile, err := setEnv(cfg.Env)

		if err != nil {
			return err
		}

		// Create .env file if it doesn't exist
		f, err := os.Create(".env")

		if err != nil {
			return errors.Wrap(err, "failed to create .env file")
		}

		defer f.Close()

		// Write envfile to .env
		_, err = f.WriteString(strings.Join(envFile, "\n"))

		if err != nil {
			return errors.Wrap(err, "failed to write envfile to .env file")
		}
	}

	for i, a := range actions {
		fmt.Print(ui.BoldBlueText(fmt.Sprintf("[%d/%d] %s", i+1, len(actions), a.Name)))
		err = a.Func(cfg)

		if err != nil {
			return errors.Wrap(err, "Failed to run action")
		}
	}

	return nil
}

func init() {
	rootValidator = validator.New()
}
