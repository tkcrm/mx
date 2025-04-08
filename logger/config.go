package logger

import (
	"fmt"
)

type Config struct {
	Format         LogFormat `yaml:"format" default:"json" usage:"allows to set custom formatting" example:"json"`
	Level          LogLevel  `yaml:"level" default:"info" usage:"allows to set custom logger level" example:"info"`
	ConsoleColored bool      `yaml:"console_colored" default:"false" usage:"allows to set colored console output" example:"false"`
	Trace          LogLevel  `yaml:"trace" default:"fatal" usage:"allows to set custom trace level" example:"fatal"`
	WithCaller     bool      `yaml:"with_caller" default:"false" usage:"allows to show caller" example:"false"`
	WithStackTrace bool      `yaml:"with_stack_trace" default:"false" usage:"allows to show stack trace" example:"false"`
}

func (c *Config) Validate() error {
	if !c.Format.Valid() {
		return fmt.Errorf("invalid logger format: %s", c.Format)
	}

	if !c.Level.Valid() {
		return fmt.Errorf("invalid logger level: %s", c.Level)
	}

	if !c.Trace.Valid() {
		return fmt.Errorf("invalid logger trace level: %s", c.Trace)
	}

	return nil
}
