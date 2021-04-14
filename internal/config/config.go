package config

import (
	"fmt"
	"regexp"
	"time"

	neopb "neo/lib/genproto/neo"
)

type Config struct {
	PingEvery    time.Duration
	RunEvery     time.Duration
	Timeout      time.Duration
	FarmURL      string
	FarmPassword string
	FlagRegexp   *regexp.Regexp
	Environ      []string
}

func ToProto(c *Config) *neopb.Config {
	return &neopb.Config{
		RunEvery:     c.RunEvery.String(),
		Timeout:      c.Timeout.String(),
		FarmUrl:      c.FarmURL,
		FarmPassword: c.FarmPassword,
		FlagRegexp:   c.FlagRegexp.String(),
		PingEvery:    c.PingEvery.String(),
		Environ:      c.Environ,
	}
}

func FromProto(config *neopb.Config) (*Config, error) {
	var (
		cfg Config
		err error
	)
	if cfg.FlagRegexp, err = regexp.Compile(config.GetFlagRegexp()); err != nil {
		return nil, fmt.Errorf("compiling regex: %w", err)
	}
	if cfg.PingEvery, err = time.ParseDuration(config.GetPingEvery()); err != nil {
		return nil, fmt.Errorf("parsing ping interval: %w", err)
	}
	if cfg.RunEvery, err = time.ParseDuration(config.GetRunEvery()); err != nil {
		return nil, fmt.Errorf("parsing run interval: %w", err)
	}
	if cfg.Timeout, err = time.ParseDuration(config.GetTimeout()); err != nil {
		return nil, fmt.Errorf("parsing timeout: %w", err)
	}
	cfg.FarmURL = config.GetFarmUrl()
	cfg.FarmPassword = config.GetFarmPassword()
	cfg.Environ = config.GetEnviron()
	return &cfg, nil
}
