package config

import (
	"fmt"
	"os"

	"github.com/a8m/envsubst"
	"github.com/goccy/go-yaml"
)

type Config struct {
	Dirs     *Dirs              `yaml:"dirs"`
	Hostname string             `yaml:"hostname"`
	Services map[string]Service `yaml:"services"`
	Alerts   []Alert            `yaml:"alerts"`
}

type Dirs struct {
	Checks string `yaml:"checks"`
	Cache  string `yaml:"cache"`
	Logs   string `yaml:"logs"`
}

type Service struct {
	URL    string            `yaml:"url"`
	Params map[string]string `yaml:"params"`
}

type Alert struct {
	Name     string         `yaml:"name"`
	Check    string         `yaml:"check"`
	SHA256   SHA256         `yaml:"sha256"`
	Trigger  Trigger        `yaml:"trigger"`
	Timeout  string         `yaml:"timeout"`
	Args     map[string]any `yaml:"args"`
	Cooldown Cooldown       `yaml:"cooldown"`
	Template string         `yaml:"template"`
	Notify   []NotifyTarget `yaml:"notify"`
}

type Trigger struct {
	Interval string `yaml:"interval"`
	Cron     string `yaml:"cron"`
	Watch    string `yaml:"watch"`
}

// SHA256 handles both string hashes and `false` (opt-out).
type SHA256 struct {
	Hash     string
	Disabled bool
}

func (s *SHA256) UnmarshalYAML(unmarshal func(any) error) error {
	var b bool
	if err := unmarshal(&b); err == nil {
		if b {
			return fmt.Errorf("sha256: true is not valid, use a hash string or false")
		}
		s.Disabled = true
		return nil
	}

	var str string
	if err := unmarshal(&str); err != nil {
		return fmt.Errorf("sha256: must be a hex string or false")
	}
	s.Hash = str
	return nil
}

// Cooldown handles a simple duration string or per-status object.
type Cooldown struct {
	Simple   string
	Warning  string
	Critical string
	Recovery bool
}

func (c *Cooldown) UnmarshalYAML(unmarshal func(any) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		c.Simple = str
		return nil
	}

	var obj cooldownObj
	if err := unmarshal(&obj); err != nil {
		return fmt.Errorf("cooldown: must be a duration string or an object with warning/critical/recovery")
	}
	c.Warning = obj.Warning
	c.Critical = obj.Critical
	c.Recovery = obj.Recovery
	return nil
}

type cooldownObj struct {
	Warning  string `yaml:"warning"`
	Critical string `yaml:"critical"`
	Recovery bool   `yaml:"recovery"`
}

// NotifyTarget handles a plain service name string or an object with overrides.
type NotifyTarget struct {
	Service  string            `yaml:"service"`
	Template string            `yaml:"template"`
	Params   map[string]string `yaml:"params"`
}

func (n *NotifyTarget) UnmarshalYAML(unmarshal func(any) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		n.Service = str
		return nil
	}

	type notifyAlias NotifyTarget
	var obj notifyAlias
	if err := unmarshal(&obj); err != nil {
		return fmt.Errorf("notify: must be a service name string or an object with service/template/params")
	}
	*n = NotifyTarget(obj)
	return nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	data, err = envsubst.Bytes(data)
	if err != nil {
		return nil, fmt.Errorf("expanding env vars: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}
