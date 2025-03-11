package cfg

import (
	"context"
	"slices"

	"github.com/cristalhq/aconfig"
	"github.com/go-playground/validator/v10"
)

type Option func(*options)

type ValidateFn struct {
	Tag                      string
	Fn                       validator.Func
	CallValidationEvenIfNull []bool
}

type Config = aconfig.Config

type options struct {
	loaderConfig  aconfig.Config
	validate      bool
	ctx           context.Context //nolint:containedctx
	validateFuncs []ValidateFn
}

func newOptions(opts ...Option) *options {
	opt := &options{
		validate:      true,
		ctx:           context.Background(),
		validateFuncs: make([]ValidateFn, 0),
	}

	for _, o := range opts {
		o(opt)
	}

	return opt
}

// WithEnvFile - path to dotenv config file.
func WithLoaderConfig(v Config) Option {
	return func(o *options) {
		o.loaderConfig = v
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
