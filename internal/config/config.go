package config

import (
	"bytes"
	"fmt"
	"os"

	"github.com/a8m/envsubst"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
)

type Config struct {
	Options  Options            `yaml:"options"`
	Globals  map[string]any     `yaml:"globals,omitempty"`
	Services map[string]Service `yaml:"services,omitempty" validate:"dive"`
	Alerts   []Alert            `yaml:"alerts,omitempty"   validate:"dive"`
}

type Options struct {
	HealthchecksDir string `yaml:"healthchecks_dir,omitempty"`
	CacheDir        string `yaml:"cache_dir,omitempty"`
	LogsDir         string `yaml:"logs_dir,omitempty"`
}

type Service struct {
	URL    string            `yaml:"url"    validate:"required"`
	Params map[string]string `yaml:"params,omitempty"`
}

type Alert struct {
	Name        string         `yaml:"name"        validate:"required"`
	Healthcheck string         `yaml:"healthcheck" validate:"required"`
	SHA256      SHA256         `yaml:"sha256,omitempty"`
	Trigger     Trigger        `yaml:"trigger"`
	Timeout     string         `yaml:"timeout,omitempty"`
	Args        map[string]any `yaml:"args,omitempty"`
	Cooldown    Cooldown       `yaml:"cooldown,omitempty"`
	Template    string         `yaml:"template"    validate:"required"`
	Notify      []NotifyTarget `yaml:"notify"      validate:"required,dive"`
}

type Trigger struct {
	Interval string `yaml:"interval,omitempty"`
	Cron     string `yaml:"cron,omitempty"`
	Watch    string `yaml:"watch,omitempty"`
	Pipe     string `yaml:"pipe,omitempty"`
}

// SHA256 handles both string hashes and `false` (opt-out).
type SHA256 struct {
	Hash     string
	Disabled bool
}

func (s SHA256) MarshalYAML() (any, error) {
	if s.Disabled {
		return false, nil
	}
	if s.Hash != "" {
		return s.Hash, nil
	}
	return nil, nil
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

func (c Cooldown) MarshalYAML() (any, error) {
	if c.Simple != "" {
		return c.Simple, nil
	}
	if c.Warning != "" || c.Critical != "" || c.Recovery {
		return cooldownObj{
			Warning:  c.Warning,
			Critical: c.Critical,
			Recovery: c.Recovery,
		}, nil
	}
	return nil, nil
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
	Service  string            `yaml:"service"  validate:"required"`
	Template string            `yaml:"template"`
	Params   map[string]string `yaml:"params"`
}

func (n NotifyTarget) MarshalYAML() (any, error) {
	if n.Template == "" && len(n.Params) == 0 {
		return n.Service, nil
	}
	type notifyAlias NotifyTarget
	return notifyAlias(n), nil
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

// LoadRaw decodes YAML without envsubst or validation.
// Used by `init --from` to load base configs that contain ${...} references.
func LoadRaw(data []byte) (*Config, error) {
	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data), yaml.Strict())
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return &cfg, nil
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

	validate := validator.New(validator.WithRequiredStructEnabled())

	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data), yaml.Validator(validate), yaml.Strict())
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	return &cfg, nil
}
