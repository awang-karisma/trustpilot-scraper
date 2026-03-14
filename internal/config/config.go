package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds configuration for the CLI scraper
type Config struct {
	TrustpilotURL string `mapstructure:"TRUSTPILOT_URL"`
	DatabaseURL   string `mapstructure:"DATABASE_URL"`
}

// ServiceConfig holds configuration for the always-running service
type ServiceConfig struct {
	// Trustpilot URL(s) for initial website setup
	TrustpilotURL string `mapstructure:"TRUSTPILOT_URL"` // Can be multiple URLs separated by semicolon

	// Migration
	DropTablesOnStart bool `mapstructure:"DROP_TABLES_ON_START"` // Safety flag for dropping tables on startup

	// Database
	DatabaseURL string `mapstructure:"DATABASE_URL"`

	// Scheduler
	DefaultSchedule string `mapstructure:"DEFAULT_SCHEDULE"` // e.g., "0 * * * *" (hourly)

	// Worker Pool
	WorkerCount      int `mapstructure:"WORKER_COUNT"`   // default: 3
	QueueSize        int `mapstructure:"QUEUE_SIZE"`     // default: 100
	ScrapeTimeoutSec int `mapstructure:"SCRAPE_TIMEOUT"` // seconds, default: 120

	// Parallel Scraping
	MaxParallelPages int `mapstructure:"MAX_PARALLEL_PAGES"` // default: 50

	// Retry
	MaxRetries      int `mapstructure:"MAX_RETRIES"`   // default: 3
	RetryBackoffSec int `mapstructure:"RETRY_BACKOFF"` // seconds, default: 5

	// Template Configuration
	TemplateDir string `mapstructure:"TEMPLATE_DIR"` // e.g., "/app/templates"

	// API
	APIEnabled bool   `mapstructure:"API_ENABLED"` // default: true
	APIPort    int    `mapstructure:"API_PORT"`    // default: 8080
	APIHost    string `mapstructure:"API_HOST"`    // default: "0.0.0.0"

	// Graceful Shutdown
	ShutdownTimeoutSec int `mapstructure:"SHUTDOWN_TIMEOUT"` // seconds, default: 30
}

// RodConfig holds browser configuration for Rod scraper
type RodConfig struct {
	BrowserPath        string `mapstructure:"CHROME_BROWSER_PATH"`
	RemoteURL          string `mapmapstructure:"CHROME_REMOTE_URL"`
	LauncherManagerURL string `mapstructure:"CHROME_LAUNCHER_MANAGER_URL"`
	Headless           bool   `mapstructure:"CHROME_HEADLESS"`
	WindowWidth        int    `mapstructure:"CHROME_WINDOW_WIDTH"`
	WindowHeight       int    `mapstructure:"CHROME_WINDOW_HEIGHT"`
	PageLoadTimeout    int    `mapstructure:"CHROME_PAGE_LOAD_TIMEOUT"`
	RetryAttempts      int    `mapstructure:"CHROME_RETRY_ATTEMPTS"`
	RetryDelay         int    `mapstructure:"CHROME_RETRY_DELAY"`
	UserAgent          string `mapstructure:"CHROME_USER_AGENT"`
	DisableImages      bool   `mapstructure:"CHROME_DISABLE_IMAGES"`
	DisableJavaScript  bool   `mapstructure:"CHROME_DISABLE_JAVASCRIPT"`
}

func LoadConfig() (*Config, error) {
	// Load .env if it exists
	_ = godotenv.Load()

	viper.AutomaticEnv()

	// Default values
	viper.SetDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/trustpilot?sslmode=disable")
	viper.SetDefault("WEBHOOK_TYPE", "custom")
	viper.SetDefault("WEBHOOK_TEMPLATE_PATH", "/app/defaults/discord.json")

	// Bind env vars to match struct tags if they don't follow the direct mapping
	viper.BindEnv("TRUSTPILOT_URL")
	viper.BindEnv("DATABASE_URL")
	viper.BindEnv("WEBHOOK_URL")
	viper.BindEnv("WEBHOOK_TYPE")
	viper.BindEnv("WEBHOOK_TEMPLATE_PATH")

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.TrustpilotURL == "" {
		return nil, fmt.Errorf("TRUSTPILOT_URL is required")
	}

	return &cfg, nil
}

// LoadServiceConfig loads configuration for the always-running service
func LoadServiceConfig() (*ServiceConfig, error) {
	// Load .env if it exists
	_ = godotenv.Load()

	viper.AutomaticEnv()

	// Default values
	viper.SetDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/trustpilot?sslmode=disable")
	viper.SetDefault("DEFAULT_SCHEDULE", "0 * * * *") // Hourly by default
	viper.SetDefault("DROP_TABLES_ON_START", false)   // Safety: don't drop tables by default
	viper.SetDefault("WORKER_COUNT", 3)
	viper.SetDefault("QUEUE_SIZE", 100)
	viper.SetDefault("SCRAPE_TIMEOUT", 120)
	viper.SetDefault("MAX_RETRIES", 3)
	viper.SetDefault("RETRY_BACKOFF", 5)
	viper.SetDefault("TEMPLATE_DIR", "/app/templates")
	viper.SetDefault("API_ENABLED", true)
	viper.SetDefault("API_PORT", 8080)
	viper.SetDefault("API_HOST", "0.0.0.0")
	viper.SetDefault("SHUTDOWN_TIMEOUT", 30)
	viper.SetDefault("MAX_PARALLEL_PAGES", 50)

	// Bind env vars
	viper.BindEnv("TRUSTPILOT_URL")
	viper.BindEnv("DROP_TABLES_ON_START")
	viper.BindEnv("DATABASE_URL")
	viper.BindEnv("DEFAULT_SCHEDULE")
	viper.BindEnv("WORKER_COUNT")
	viper.BindEnv("QUEUE_SIZE")
	viper.BindEnv("SCRAPE_TIMEOUT")
	viper.BindEnv("MAX_RETRIES")
	viper.BindEnv("RETRY_BACKOFF")
	viper.BindEnv("TEMPLATE_DIR")
	viper.BindEnv("API_ENABLED")
	viper.BindEnv("API_PORT")
	viper.BindEnv("API_HOST")
	viper.BindEnv("SHUTDOWN_TIMEOUT")
	viper.BindEnv("MAX_PARALLEL_PAGES")

	var cfg ServiceConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// NewRodConfig creates a default Rod configuration
func NewRodConfig() *RodConfig {
	return &RodConfig{
		Headless:          true,
		WindowWidth:       1920,
		WindowHeight:      1080,
		PageLoadTimeout:   30,
		RetryAttempts:     3,
		RetryDelay:        2,
		UserAgent:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		DisableImages:     true,
		DisableJavaScript: false,
	}
}

// LoadRodConfig loads Rod configuration from environment variables
func LoadRodConfig() *RodConfig {
	cfg := NewRodConfig()

	// Load from environment variables with viper
	cfg.BrowserPath = viper.GetString("CHROME_BROWSER_PATH")
	cfg.RemoteURL = viper.GetString("CHROME_REMOTE_URL")
	cfg.LauncherManagerURL = viper.GetString("CHROME_LAUNCHER_MANAGER_URL")
	cfg.Headless = viper.GetBool("CHROME_HEADLESS")
	cfg.WindowWidth = viper.GetInt("CHROME_WINDOW_WIDTH")
	cfg.WindowHeight = viper.GetInt("CHROME_WINDOW_HEIGHT")
	cfg.PageLoadTimeout = viper.GetInt("CHROME_PAGE_LOAD_TIMEOUT")
	cfg.RetryAttempts = viper.GetInt("CHROME_RETRY_ATTEMPTS")
	cfg.RetryDelay = viper.GetInt("CHROME_RETRY_DELAY")
	cfg.UserAgent = viper.GetString("CHROME_USER_AGENT")
	cfg.DisableImages = viper.GetBool("CHROME_DISABLE_IMAGES")
	cfg.DisableJavaScript = viper.GetBool("CHROME_DISABLE_JAVASCRIPT")

	return cfg
}
