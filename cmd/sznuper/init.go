package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/sznuper/sznuper/internal/config"
	"github.com/sznuper/sznuper/internal/initcmd"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new sznuper configuration",
	Long:  "Interactive TUI or non-interactive config generator. Use --add-service for non-interactive mode.",
	RunE:  runInit,
}

var (
	initFrom       string
	initAddService []string
	initForce      bool
	initOutput     string
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&initFrom, "from", "", "path or URL to base config")
	initCmd.Flags().StringArrayVar(&initAddService, "add-service", nil, "add service as name:shoutrrr-url (repeatable)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing config file")
	initCmd.Flags().StringVarP(&initOutput, "output", "o", "", "output path (overrides auto-detect)")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine output path
	outPath := initOutput
	if outPath == "" {
		outPath = config.DefaultWritePath()
	}

	// Check existing file
	if !initForce {
		if _, err := os.Stat(outPath); err == nil {
			return fmt.Errorf("config already exists at %s (use --force to overwrite)", outPath)
		}
	}

	// Load base config or start fresh
	cfg, err := loadBaseConfig()
	if err != nil {
		return err
	}

	// Fill defaults for missing fields
	applyDefaults(cfg)

	// Decide mode: non-interactive if --add-service present
	if len(initAddService) > 0 {
		return runNonInteractive(cfg, outPath)
	}

	// Interactive mode — require a TTY
	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		return fmt.Errorf("no TTY detected and no --add-service flags; cannot run interactive mode")
	}

	return initcmd.Run(cfg, outPath)
}

func loadBaseConfig() (*config.Config, error) {
	if initFrom == "" {
		return initcmd.DefaultConfig()
	}

	data, err := readSource(initFrom)
	if err != nil {
		return nil, fmt.Errorf("loading base config: %w", err)
	}

	cfg, err := config.LoadRaw(data)
	if err != nil {
		return nil, fmt.Errorf("parsing base config: %w", err)
	}
	return cfg, nil
}

func readSource(src string) ([]byte, error) {
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		resp, err := http.Get(src)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, src)
		}
		return io.ReadAll(resp.Body)
	}
	return os.ReadFile(src)
}

func applyDefaults(cfg *config.Config) {
	defaults := config.DefaultOptions()
	if cfg.Options.HealthchecksDir == "" {
		cfg.Options.HealthchecksDir = defaults.HealthchecksDir
	}
	if cfg.Options.CacheDir == "" {
		cfg.Options.CacheDir = defaults.CacheDir
	}
	if cfg.Options.LogsDir == "" {
		cfg.Options.LogsDir = defaults.LogsDir
	}

	if cfg.Globals == nil {
		cfg.Globals = config.DefaultGlobals()
	} else if _, ok := cfg.Globals["hostname"]; !ok {
		dg := config.DefaultGlobals()
		cfg.Globals["hostname"] = dg["hostname"]
	}
}

func runNonInteractive(cfg *config.Config, outPath string) error {
	if cfg.Services == nil {
		cfg.Services = make(map[string]config.Service)
	}

	for _, entry := range initAddService {
		name, url, ok := strings.Cut(entry, ":")
		if !ok || name == "" || url == "" {
			return fmt.Errorf("invalid --add-service format %q (expected name:shoutrrr-url)", entry)
		}
		if _, exists := cfg.Services[name]; exists {
			return fmt.Errorf("service %q already exists in config", name)
		}
		cfg.Services[name] = config.Service{URL: url}
	}

	if err := config.Write(cfg, outPath); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Config written to %s\n", outPath)
	return nil
}
