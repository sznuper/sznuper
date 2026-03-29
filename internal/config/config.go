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
	Channels map[string]Channel `yaml:"channels,omitempty" validate:"dive"`
	Alerts   []Alert            `yaml:"alerts,omitempty"   validate:"dive"`
}

type Options struct {
	HealthchecksDir string `yaml:"healthchecks_dir,omitempty"`
	CacheDir        string `yaml:"cache_dir,omitempty"`
	LogsDir         string `yaml:"logs_dir,omitempty"`
}

type Channel struct {
	URL    string            `yaml:"url"    validate:"required"`
	Params map[string]string `yaml:"params,omitempty"`
}

type Alert struct {
	Name        string         `yaml:"name"        validate:"required"`
	Healthcheck string         `yaml:"healthcheck" validate:"required"`
	SHA256      SHA256         `yaml:"sha256,omitempty"`
	Triggers    []Trigger      `yaml:"triggers"`
	Timeout     string         `yaml:"timeout,omitempty"`
	Args        map[string]any `yaml:"args,omitempty"`
	SideEffects []string       `yaml:"side_effects,omitempty"`
	Template    string         `yaml:"template"    validate:"required"`
	Cooldown    string         `yaml:"cooldown,omitempty"`
	Notify      []NotifyTarget `yaml:"notify,omitempty" validate:"dive"`
	Events      *Events        `yaml:"events,omitempty"`
}

type Trigger struct {
	Interval  string `yaml:"interval,omitempty"`
	Cron      string `yaml:"cron,omitempty"`
	Watch     string `yaml:"watch,omitempty"`
	Pipe      string `yaml:"pipe,omitempty"`
	Lifecycle bool   `yaml:"lifecycle,omitempty"`
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

// Events configures per-event-type handling for an alert.
type Events struct {
	Healthy     []string                 `yaml:"healthy,omitempty"`
	OnUnmatched string                   `yaml:"on_unmatched,omitempty"`
	Override    map[string]EventOverride `yaml:"override,omitempty"`
}

// EventOverride provides per-event-type overrides for template, cooldown, and notify.
type EventOverride struct {
	Template string         `yaml:"template,omitempty"`
	Cooldown string         `yaml:"cooldown,omitempty"`
	Notify   []NotifyTarget `yaml:"notify,omitempty"`
}

// NotifyTarget handles a plain channel name string or a channel object with params.
//
// YAML formats:
//
//   - telegram                    (plain string)
//   - telegram:                   (map with channel name as key)
//     params:
//     notification: "false"
type NotifyTarget struct {
	Channel string            `yaml:"-" validate:"required"`
	Params  map[string]string `yaml:"params,omitempty"`
}

func (n NotifyTarget) MarshalYAML() (any, error) {
	if len(n.Params) == 0 {
		return n.Channel, nil
	}
	type paramsOnly struct {
		Params map[string]string `yaml:"params"`
	}
	return map[string]any{
		n.Channel: paramsOnly{Params: n.Params},
	}, nil
}

func (n *NotifyTarget) UnmarshalYAML(unmarshal func(any) error) error {
	// Try plain string: "telegram"
	var str string
	if err := unmarshal(&str); err == nil {
		n.Channel = str
		return nil
	}

	// Try map: { telegram: { params: { ... } } }
	var obj map[string]struct {
		Params map[string]string `yaml:"params"`
	}
	if err := unmarshal(&obj); err != nil {
		return fmt.Errorf("notify: must be a channel name string or a channel object")
	}
	if len(obj) != 1 {
		return fmt.Errorf("notify: channel object must have exactly one key")
	}
	for name, cfg := range obj {
		n.Channel = name
		n.Params = cfg.Params
	}
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
