package cfg

import (
	"fmt"
	"os"
	"path"
	"reflect"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigdotenv"
)

// LoadForTests environment variables from `os env`, `.env` file and pass it to struct for tests.
//
// Disabled flags detection and any cli features.
//
// For local development use `.env` file from root project.
//
// LoadForTests also call a `Validate` method.
//
// Example:
//
//	var config internalConfig.Config
//	if err := cfg.Load(&config); err != nil {
//		log.Fatalf("could not load configuration: %v", err)
//	}
func LoadForTests(cfg any, opts ...Option) error {
	if reflect.ValueOf(cfg).Kind() != reflect.Ptr {
		return fmt.Errorf("config must be a pointer")
	}

	options := newOptions(opts...)

	if options.envPath == "" {
		pwdDir, err := os.Getwd()
		if err != nil {
			return err
		}
		options.envPath = pwdDir
	}

	c := config{
		out:     os.Stdout,
		exit:    os.Exit,
		args:    os.Args[1:],
		options: options,
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

	if options.validate {
		if err := c.validateEnvs(cfg, loader); err != nil {
			return err
		}
	}

	return nil
}
