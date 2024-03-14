package cfg

import (
	"context"
	"errors"
	"flag"
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
	"github.com/go-playground/validator/v10"
	"github.com/tkcrm/mx/util/files"
	"github.com/tkcrm/mx/util/structs"
)

var (
	boolTrueValues = []string{"true", "1"}
	fileDecoders   = map[string]aconfig.FileDecoder{
		".env":  aconfigdotenv.New(),
		".yaml": aconfigyaml.New(),
	}
)

type config struct {
	options *options

	filePath string
	out      io.Writer

	args []string
	exit func(int)

	showHelp bool
	showCurr bool
	validate bool
	markdown bool
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

	// validate options
	if options.envFile == "" && options.yamlFile == "" {
		return fmt.Errorf("env file or yaml file must be provided")
	}

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

	flags := loader.Flags()
	flags.SetOutput(c.out)
	flags.Usage = func() { c.renderHelp(loader, flags) }

	c.attachFlags(flags)

	if err := flags.Parse(c.args); err != nil && !errors.Is(err, flag.ErrHelp) {
		return fmt.Errorf("could not parse flags: %w", err)
	}

	if err := loader.Load(); err != nil {
		return err
	}

	switch {
	default:
	case c.showCurr:
		// on version requested
		c.print("Version: " + options.version)

		c.exit(0)

	case c.markdown:
		// on markdown requested
		c.generateMarkdown(loader)

		c.exit(0)

	case c.showHelp:
		// on help requested
		c.renderHelp(loader, flags)

		c.exit(0)

	case c.validate:
		// on validate requested
		if err := c.validateEnvs(cfg, loader); err != nil {
			fmt.Println(err)
			c.exit(2)
		}

		c.print("OK")

		c.exit(0)
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

func (c *config) validateEnvs(cfg any, loader *aconfig.Loader) error {
	if err := validateElem(c.options.ctx, cfg); err != nil {
		return err
	}

	val := reflect.ValueOf(cfg).Elem()
	for i := 0; i < val.NumField(); i++ {
		if err := validateElem(c.options.ctx, val.Field(i).Addr().Interface()); err != nil {
			return err
		}
	}

	// init validator
	validate := validator.New()
	for _, item := range c.options.validateFuncs {
		validate.RegisterValidation(item.Tag, item.Fn, item.CallValidationEvenIfNull...)
	}

	// validate struct
	errs := []string{}
	configFields := getConfigFields(loader)
	for _, f := range configFields {
		if f.disableValidation || f.validateParams == "" {
			continue
		}

		fieldValue, err := structs.LookupString(cfg, f.path)
		if err != nil {
			return err
		}

		if err := validate.Var(fieldValue.Interface(), f.validateParams); err != nil {
			errs = append(errs,
				strings.ReplaceAll(
					err.Error(),
					"Key: '' Error:Field validation for ''",
					fmt.Sprintf("Validate %s env error:", f.envName),
				),
			)
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

func validateElem(ctx context.Context, elem any) error {
	// try to validate with Validate() error
	if tmp, ok := elem.(interface {
		Validate() error
	}); ok {
		if err := tmp.Validate(); err != nil {
			return err
		}
	}

	// try to validate with Validate(ctx context.Context) error
	if ctx != nil {
		if tmp, ok := elem.(interface {
			Validate(ctx context.Context) error
		}); ok {
			if err := tmp.Validate(ctx); err != nil {
				return err
			}
		}
	}

	return nil
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

		if slices.Contains(boolTrueValues, strings.ToLower(f.Tag("disableValidation"))) {
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

	aconf := aconfig.Config{
		AllowUnknownFields: conf.options.allowUnknownFields,
		SkipFlags:          conf.options.skipFlags,
		FileDecoders:       fileDecoders,
	}

	if conf.options.yamlFile != "" {
		yamlFile := path.Join(pwdDir, conf.options.yamlFile)
		if files.ExistsPath(yamlFile) {
			aconf.Files = append(aconf.Files, yamlFile)
		}
	}

	if conf.options.envFile != "" {
		dotEnvFile := path.Join(pwdDir, conf.options.envFile)
		if files.ExistsPath(dotEnvFile) {
			aconf.Files = append(aconf.Files, dotEnvFile)
		}
	}

	return aconf, nil
}
