package cfg

import (
	"fmt"
	"os"
	"path"
	"reflect"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigdotenv"
)

type IConfig interface {
	Validate() error
}

// LoadConfig - load environment variables from `os env`, `.env` file and pass it to struct.
//
// For local development use `.env` file from root project.
//
// LoadConfig also call a `Validate` method.
//
// Example:
//
//	var config internalConfig.Config
//	if err := cfg.LoadConfig(&config); err != nil {
//		log.Fatalf("could not load configuration: %v", err)
//	}
func LoadConfig(cfg IConfig, opts ...Option) error {
	if reflect.ValueOf(cfg).Kind() != reflect.Ptr {
		return fmt.Errorf("config variable must be a pointer")
	}

	options := options{
		envFile: ".env",
	}
	for _, opt := range opts {
		opt(&options)
	}

	if options.envPath == "" {
		pwdDir, err := os.Getwd()
		if err != nil {
			return err
		}
		options.envPath = pwdDir
	}

	aconf := aconfig.Config{
		AllowUnknownFields: true,
		SkipFlags:          true,
		Files:              []string{path.Join(options.envPath, options.envFile)},
		FileDecoders: map[string]aconfig.FileDecoder{
			options.envFile: aconfigdotenv.New(),
		},
	}

	loader := aconfig.LoaderFor(cfg, aconf)
	if err := loader.Load(); err != nil {
		return err
	}

	return cfg.Validate()
}
