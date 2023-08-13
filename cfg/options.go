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
	version       string
	envFile       string
	envPath       string
	validate      bool
	ctx           context.Context
	validateFuncs []ValidateFn
}

func newOptions(opts ...Option) *options {
	opt := &options{
		envFile:       ".env",
		validate:      true,
		ctx:           context.Background(),
		validateFuncs: make([]ValidateFn, 0),
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

func WithEnvFile(v string) Option {
	return func(o *options) {
		o.envFile = v
	}
}

func WithEnvPath(v string) Option {
	return func(o *options) {
		o.envPath = v
	}
}

func WithValidate(v bool) Option {
	return func(o *options) {
		o.validate = v
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
