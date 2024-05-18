package cfg

import (
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigdotenv"
	"github.com/cristalhq/aconfig/aconfigyaml"
	"github.com/tkcrm/mx/util/files"
)

var (
	boolTrueValues = []string{"true", "1"}
	fileDecoders   = map[string]aconfig.FileDecoder{
		".env":  aconfigdotenv.New(),
		".yaml": aconfigyaml.New(),
		".yml":  aconfigyaml.New(),
	}
)

type config struct {
	options *options

	out io.Writer

	args []string
	exit func(int)
}

// Load environment variables from `os env`, flags, `.env`, `.yaml` files and pass it to struct.
//
// For local development use `.env` file from root project.
//
// Load also call a `Validate` method if it proided.
//
// Example:
//
//	var config internalConfig.Config
//	if err := cfg.Load(&config); err != nil {
//		logger.Fatalf("could not load configuration: %v", err)
//	}
func Load(cfg any, opts ...Option) error {
	if reflect.ValueOf(cfg).Kind() != reflect.Ptr {
		return fmt.Errorf("config must be a pointer")
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

	if options.validate {
		if err := c.validateEnvs(cfg, loader); err != nil {
			return err
		}
	}

	return nil
}

func (c *config) print(value string) {
	_, _ = fmt.Fprintln(c.out, value)
}

func getConfigFields(loader *aconfig.Loader) []configField {
	res := []configField{}

	loader.WalkFields(func(f aconfig.Field) bool {
		newField := configField{
			path:           f.Name(),
			defaultValue:   f.Tag("default"),
			usage:          f.Tag("usage"),
			example:        f.Tag("example"),
			validateParams: f.Tag("validate"),
		}

		if slices.Contains(boolTrueValues, strings.ToLower(f.Tag("required"))) {
			newField.isRequired = true
		}

		if slices.Contains(boolTrueValues, strings.ToLower(f.Tag("secret"))) {
			newField.isSecret = true
		}

		if slices.Contains(boolTrueValues, strings.ToLower(f.Tag("disable_validation"))) {
			newField.disableValidation = true
		}

		if strings.Contains(newField.validateParams, "required") {
			newField.isRequired = true
		}

		envName := f.Tag("env")

		field := f
		var ok bool
		for {
			if field, ok = field.Parent(); !ok {
				break
			}

			envName = fmt.Sprintf("%s_%s", field.Tag("env"), envName)

			if !newField.disableValidation &&
				slices.Contains(
					boolTrueValues,
					strings.ToLower(field.Tag("disable_validation")),
				) {
				newField.disableValidation = true
			}
		}

		newField.envName = envName

		res = append(res, newField)

		return true
	})

	return res
}

// getAconfig return aconfig.Config based on options
func getAconfig(conf config) (aconfig.Config, error) {
	pwdDir, err := os.Getwd()
	if err != nil {
		return aconfig.Config{}, err
	}

	aconf := conf.options.loaderConfig
	aconf.FileDecoders = fileDecoders

	for _, file := range aconf.Files {
		fpath := path.Join(pwdDir, file)
		if !files.ExistsPath(fpath) {
			return aconfig.Config{}, fmt.Errorf("config file not found: %s", fpath)
		}
	}

	return aconf, nil
}
