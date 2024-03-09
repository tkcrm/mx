package cfg

import (
	"context"
	"slices"

	"github.com/go-playground/validator/v10"
)

type Option func(*options)

type ValidateFn struct {
	Tag                      string
	Fn                       validator.Func
	CallValidationEvenIfNull []bool
}

type options struct {
	version            string
	envFile            string
	yamlFile           string
	validate           bool
	skipFlags          bool
	allowUnknownFields bool
	ctx                context.Context
	validateFuncs      []ValidateFn
}

func newOptions(opts ...Option) *options {
	opt := &options{
		envFile:            ".env",
		validate:           true,
		skipFlags:          true,
		allowUnknownFields: true,
		ctx:                context.Background(),
		validateFuncs:      make([]ValidateFn, 0),
	}

	for _, o := range opts {
		o(opt)
	}

	return opt
}

func WithVersion(v string) Option {
	return func(o *options) {
		o.version = v
	}
}

// WithEnvFile - path to don env config file
func WithEnvFile(v string) Option {
	return func(o *options) {
		o.envFile = v
	}
}

// WithYamlFile path to yaml config file
func WithYamlFile(v string) Option {
	return func(o *options) {
		o.yamlFile = v
	}
}

func WithValidate(v bool) Option {
	return func(o *options) {
		o.validate = v
	}
}

// WithSkipFlags - skip flags parsing
//
// By default it true
func WithSkipFlags(v bool) Option {
	return func(o *options) {
		o.skipFlags = v
	}
}

// WithAllowUnknownFields - allow unknown fields in config
//
// By default it true
func WithAllowUnknownFields(v bool) Option {
	return func(o *options) {
		o.allowUnknownFields = v
	}
}

func WithContext(v context.Context) Option {
	return func(o *options) {
		o.ctx = v
	}
}

func WithValidateFuncs(items ...ValidateFn) Option {
	return func(o *options) {
		for _, item := range items {
			if item.Tag == "" ||
				item.Fn == nil {
				continue
			}

			if ok := slices.ContainsFunc(o.validateFuncs, func(el ValidateFn) bool {
				return el.Tag == item.Tag
			}); ok {
				continue
			}

			o.validateFuncs = append(o.validateFuncs, item)
		}
	}
}
