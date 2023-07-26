package logger

import validation "github.com/go-ozzo/ozzo-validation/v4"

type Config struct {
	EncodingConsole bool   `json:"LOG_ENCODING_CONSOLE" env:"ENCODING_CONSOLE" default:"false" usage:"allows to set user-friendly formatting"`
	Level           string `json:"LOG_LEVEL" env:"LEVEL" default:"info" usage:"allows to set custom logger level"`
	Trace           string `json:"LOG_TRACE" env:"TRACE" default:"fatal" usage:"allows to set custom trace level"`
	WithCaller      bool   `json:"LOG_WITH_CALLER" env:"WITH_CALLER" default:"false" usage:"allows to show stack trace"`
	WithStackTrace  bool   `json:"LOG_WITH_STACK_TRACE" env:"WITH_STACK_TRACE" default:"false" usage:"allows to show stack trace"`
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
