package projectconfig

import (
	"fmt"
	"os"

	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/types"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

var rootValidator *validator.Validate

func init() {
	rootValidator = validator.New()
}

func LoadProjectConfig() (*types.IBLProject, error) {
	// Open pkg.yaml
	fmt.Print(ui.BoldText("[INIT] Opening project.yaml"))

	bytes, err := os.ReadFile("project.yaml")

	if err != nil {
		return nil, err
	}

	// Parse project.yaml
	var proj types.IBLProject

	err = yaml.Unmarshal(bytes, &proj)

	if err != nil {
		return nil, err
	}

	// Check if the proj is valid
	err = rootValidator.Struct(proj)

	if err != nil {
		return nil, err
	}

	return &proj, nil
}
