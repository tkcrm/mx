package logger

import validation "github.com/go-ozzo/ozzo-validation/v4"

type Config struct {
	EncodingConsole bool   `yaml:"encoding_console" env:"ENCODING_CONSOLE" default:"false" usage:"allows to set user-friendly formatting" example:"false"`
	Level           string `yaml:"level" env:"LEVEL" default:"info" usage:"allows to set custom logger level" example:"info"`
	Trace           string `yaml:"trace" env:"TRACE" default:"fatal" usage:"allows to set custom trace level" example:"fatal"`
	WithCaller      bool   `yaml:"with_caller" env:"WITH_CALLER" default:"false" usage:"allows to show stack trace" example:"false"`
	WithStackTrace  bool   `yaml:"with_stack_trace" env:"WITH_STACK_TRACE" default:"false" usage:"allows to show stack trace" example:"false"`
}

func (c *Config) Validate() error {
	if err := validation.ValidateStruct(c,
		validation.Field(&c.Level, validation.Required, validation.In(allLevels...)),
		validation.Field(&c.Trace, validation.Required, validation.In(allLevels...)),
	); err != nil {
		return err
	}

	return nil
}
