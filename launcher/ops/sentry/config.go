package sentry

import (
	"github.com/getsentry/sentry-go"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Option func(*Config)

type Config struct {
	Enabled          bool    `default:"false"`
	DSN              string  `yaml:"dsn" usage:"The DSN to use. If the DSN is not set, the client is effectively disabled."`
	Environment      string  `usage:"The environment to be sent with events."`
	TracesSampleRate float64 `yaml:"traces_sample_rate" default:"1"`
	AttachStacktrace bool    `yaml:"attach_stacktrace" default:"true"`

	sentryConfig sentry.ClientOptions
	appVersion   string
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	return validation.ValidateStruct(
		c,
		validation.Field(&c.DSN, validation.Required),
		validation.Field(&c.Environment, validation.Required),
	)
}

func WithSentryConfig(v sentry.ClientOptions) Option {
	return func(c *Config) {
		c.sentryConfig = v
	}
}

func WithAppVersion(appVersion string) Option {
	return func(c *Config) {
		c.appVersion = appVersion
	}
}
