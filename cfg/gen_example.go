package cfg

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func GenerateYamlTemplate(path string, config any) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config error: %w", err)
	}

	if err := os.WriteFile(path, data, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
