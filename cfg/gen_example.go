package cfg

import (
	"fmt"
	"os"
	"reflect"

	"gopkg.in/yaml.v3"
)

func GenerateYamlTemplate(cfg any, path string) error {
	if reflect.ValueOf(cfg).Kind() != reflect.Ptr {
		return fmt.Errorf("config must be a pointer")
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config error: %w", err)
	}

	if err := os.WriteFile(path, data, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
