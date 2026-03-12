package initcmd

import "fmt"

// ServiceField describes a single input required to build a Shoutrrr URL.
type ServiceField struct {
	Label       string // "Bot Token"
	EnvVar      string // default env var name, e.g. "TELEGRAM_TOKEN"
	Placeholder string // hint text
	IsSecret    bool   // mask in TUI
	IsParam     bool   // goes into service params map, not URL
	ParamKey    string // actual Shoutrrr param key, e.g. "chats" (only if IsParam)
}

// ServiceType describes a notification service template.
type ServiceType struct {
	Name     string         // "telegram"
	Label    string         // "Telegram"
	Fields   []ServiceField // ordered inputs
	BuildURL func(vals map[string]string) string
}

// envRef wraps a name in ${...} syntax for config output.
func envRef(name string) string {
	return fmt.Sprintf("${%s}", name)
}

// Registry is the ordered list of all supported service types.
var Registry = []ServiceType{
	{
		Name:  "telegram",
		Label: "Telegram",
		Fields: []ServiceField{
			{Label: "Bot Token", EnvVar: "TELEGRAM_TOKEN", Placeholder: "e.g. 123456:ABC-DEF...", IsSecret: true},
			{Label: "Chat ID", EnvVar: "TELEGRAM_CHAT_ID", Placeholder: "e.g. -1001234567890", IsParam: true, ParamKey: "chats"},
		},
		BuildURL: func(v map[string]string) string {
			return fmt.Sprintf("telegram://%s@telegram", envRef(v["Bot Token"]))
		},
	},
	{
		Name:  "discord",
		Label: "Discord",
		Fields: []ServiceField{
			{Label: "Token", EnvVar: "DISCORD_TOKEN", Placeholder: "webhook token", IsSecret: true},
			{Label: "Webhook ID", EnvVar: "DISCORD_WEBHOOK_ID", Placeholder: "webhook ID"},
		},
		BuildURL: func(v map[string]string) string {
			return fmt.Sprintf("discord://%s@%s", envRef(v["Token"]), envRef(v["Webhook ID"]))
		},
	},
	{
		Name:  "slack",
		Label: "Slack",
		Fields: []ServiceField{
			{Label: "Token A", EnvVar: "SLACK_TOKEN_A", Placeholder: "first token part", IsSecret: true},
			{Label: "Token B", EnvVar: "SLACK_TOKEN_B", Placeholder: "second token part", IsSecret: true},
			{Label: "Token C", EnvVar: "SLACK_TOKEN_C", Placeholder: "third token part", IsSecret: true},
		},
		BuildURL: func(v map[string]string) string {
			return fmt.Sprintf("slack://%s/%s/%s", envRef(v["Token A"]), envRef(v["Token B"]), envRef(v["Token C"]))
		},
	},
	{
		Name:  "teams",
		Label: "Microsoft Teams",
		Fields: []ServiceField{
			{Label: "Group", EnvVar: "TEAMS_GROUP", Placeholder: "group UUID"},
			{Label: "Tenant", EnvVar: "TEAMS_TENANT", Placeholder: "tenant UUID"},
			{Label: "Alt ID", EnvVar: "TEAMS_ALT_ID", Placeholder: "alt ID"},
			{Label: "Group Owner", EnvVar: "TEAMS_GROUP_OWNER", Placeholder: "group owner"},
		},
		BuildURL: func(v map[string]string) string {
			return fmt.Sprintf("teams://%s@%s/%s/%s",
				envRef(v["Group"]), envRef(v["Tenant"]),
				envRef(v["Alt ID"]), envRef(v["Group Owner"]))
		},
	},
	{
		Name:  "smtp",
		Label: "Email (SMTP)",
		Fields: []ServiceField{
			{Label: "Username", EnvVar: "SMTP_USER", Placeholder: "SMTP username"},
			{Label: "Password", EnvVar: "SMTP_PASS", Placeholder: "SMTP password", IsSecret: true},
			{Label: "Host", EnvVar: "SMTP_HOST", Placeholder: "e.g. smtp.gmail.com"},
			{Label: "Port", EnvVar: "SMTP_PORT", Placeholder: "e.g. 587"},
			{Label: "From Address", EnvVar: "SMTP_FROM", Placeholder: "sender@example.com", IsParam: true, ParamKey: "fromaddress"},
			{Label: "To Address", EnvVar: "SMTP_TO", Placeholder: "recipient@example.com", IsParam: true, ParamKey: "toaddresses"},
		},
		BuildURL: func(v map[string]string) string {
			return fmt.Sprintf("smtp://%s:%s@%s:%s/",
				envRef(v["Username"]), envRef(v["Password"]),
				envRef(v["Host"]), envRef(v["Port"]))
		},
	},
	{
		Name:  "gotify",
		Label: "Gotify",
		Fields: []ServiceField{
			{Label: "Host", EnvVar: "GOTIFY_HOST", Placeholder: "e.g. gotify.example.com"},
			{Label: "Token", EnvVar: "GOTIFY_TOKEN", Placeholder: "app token", IsSecret: true},
		},
		BuildURL: func(v map[string]string) string {
			return fmt.Sprintf("gotify://%s/%s", envRef(v["Host"]), envRef(v["Token"]))
		},
	},
	{
		Name:  "ntfy",
		Label: "Ntfy",
		Fields: []ServiceField{
			{Label: "Host", EnvVar: "NTFY_HOST", Placeholder: "e.g. ntfy.sh"},
			{Label: "Topic", EnvVar: "NTFY_TOPIC", Placeholder: "topic name"},
		},
		BuildURL: func(v map[string]string) string {
			return fmt.Sprintf("ntfy://%s/%s", envRef(v["Host"]), envRef(v["Topic"]))
		},
	},
	{
		Name:  "pushover",
		Label: "Pushover",
		Fields: []ServiceField{
			{Label: "API Token", EnvVar: "PUSHOVER_TOKEN", Placeholder: "application API token", IsSecret: true},
			{Label: "User Key", EnvVar: "PUSHOVER_USER", Placeholder: "user key"},
		},
		BuildURL: func(v map[string]string) string {
			return fmt.Sprintf("pushover://shoutrrr:%s@%s", envRef(v["API Token"]), envRef(v["User Key"]))
		},
	},
	{
		Name:  "matrix",
		Label: "Matrix",
		Fields: []ServiceField{
			{Label: "Username", EnvVar: "MATRIX_USER", Placeholder: "matrix username"},
			{Label: "Password", EnvVar: "MATRIX_PASS", Placeholder: "password", IsSecret: true},
			{Label: "Host", EnvVar: "MATRIX_HOST", Placeholder: "e.g. matrix.org"},
			{Label: "Port", EnvVar: "MATRIX_PORT", Placeholder: "e.g. 443"},
		},
		BuildURL: func(v map[string]string) string {
			return fmt.Sprintf("matrix://%s:%s@%s:%s/",
				envRef(v["Username"]), envRef(v["Password"]),
				envRef(v["Host"]), envRef(v["Port"]))
		},
	},
	{
		Name:  "generic",
		Label: "Generic Webhook",
		Fields: []ServiceField{
			{Label: "URL", EnvVar: "WEBHOOK_URL", Placeholder: "e.g. example.com/webhook"},
		},
		BuildURL: func(v map[string]string) string {
			return fmt.Sprintf("generic+https://%s", envRef(v["URL"]))
		},
	},
	{
		Name:   "logger",
		Label:  "Logger",
		Fields: nil,
		BuildURL: func(v map[string]string) string {
			return "logger://"
		},
	},
	{
		Name:  "custom",
		Label: "Custom Shoutrrr URL",
		Fields: []ServiceField{
			{Label: "Shoutrrr URL", EnvVar: "", Placeholder: "e.g. telegram://token@telegram"},
		},
		BuildURL: func(v map[string]string) string {
			return v["Shoutrrr URL"]
		},
	},
}
