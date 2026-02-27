# sznuper — Dependency Decisions

## Config Loading & Validation

**Choice:** `goccy/go-yaml` + `go-playground/validator`
**Rejected:** `spf13/viper`, `kaptinlin/gozod`

### Why not Viper?

Viper is a multi-source config toolkit (YAML, TOML, HCL, env vars, remote config). Sznuper reads a single YAML file — Viper's features are unnecessary weight.

Practical problems:

- **No built-in validation.** Open issue [#702](https://github.com/spf13/viper/issues/702) was closed as not planned.
- **mapstructure indirection.** Viper decodes via `mapstructure`, not `yaml` tags. Requires triple struct tags (`yaml`, `mapstructure`, `validate`) or a `TagName` override hack.
- **Lowercases all keys silently.** Can cause subtle mismatches.
- **Large dependency tree.** Pulls in TOML, HCL, afero, etc. that Sznuper will never use.

### Why goccy/go-yaml?

- **`yaml.Strict()` / `DisallowUnknownField()`** — rejects unknown YAML keys, catching typos like `tirgger:` instead of `trigger:`.
- **Built-in `StructValidator` interface** — plugs in `go-playground/validator` at decode time. Validation errors include YAML line/column numbers.
- **Native `yaml` struct tags** — no mapstructure indirection.
- **Single-step decode+validate** — one call does parse, strict-field check, and struct validation.

### Why go-playground/validator?

- ~17k stars, battle-tested, de facto standard for Go struct validation.
- Struct tags cover all Sznuper needs: `required`, `oneof`, `min`, `max`, `dive`, plus custom validators (e.g. duration parsing).
- Satisfies goccy/go-yaml's `StructValidator` interface directly — no adapter needed.

### Why not gozod?

[kaptinlin/gozod](https://github.com/kaptinlin/gozod) is a Zod-inspired fluent validation library for Go.

- **18 stars, created mid-2025** — too early-stage, tiny community.
- **Doesn't satisfy goccy/go-yaml's `StructValidator` interface** — would lose line-number error reporting.
- **Fluent API solves a different problem** — great for runtime schema building, but Sznuper's config is a fixed struct where tags are the right tool.

### Usage Pattern

```go
validate := validator.New(validator.WithRequiredStructEnabled())

var cfg Config
dec := yaml.NewDecoder(f, yaml.Validator(validate), yaml.Strict())
if err := dec.Decode(&cfg); err != nil {
    // errors include YAML line numbers + validation failures
    return nil, fmt.Errorf("config error: %w", err)
}
```
