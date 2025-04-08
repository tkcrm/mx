package cfg

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/cristalhq/aconfig"
	"github.com/goccy/go-yaml"
)

func GenerateYamlTemplate(cfg any, filePath string, opts ...Option) error {
	if reflect.ValueOf(cfg).Kind() != reflect.Ptr {
		return errors.New("config must be a pointer")
	}

	options := newOptions(opts...)

	c := config{
		out:     os.Stdout,
		exit:    os.Exit,
		args:    os.Args[1:],
		options: options,
	}

	aconf, err := getAconfig(c)
	if err != nil {
		return err
	}

	loader := aconfig.LoaderFor(cfg, aconf)

	if err := loader.Load(); err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	enc := yaml.NewEncoder(buf, yaml.Indent(2))
	defer enc.Close()

	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode yaml: %w", err)
	}

	if err := os.WriteFile(filePath, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
