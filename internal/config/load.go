// Package config provides configuration loading for dnsmonster.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// LoadConfig loads the configuration from TOML file, environment variables, and flags.
func LoadConfig() (*Config, error) {
	var configFile string
	pflag.StringVar(&configFile, "config", "dnsmonster.toml", "Path to configuration file")
	pflag.Parse()

	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigFile(configFile)
	v.SetEnvPrefix("DNSMONSTER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set default config file search paths
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/dnsmonster/")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// Only error if file is not found and not the default
		if _, ok := err.(viper.ConfigFileNotFoundError); ok && configFile == "dnsmonster.toml" {
			fmt.Fprintf(os.Stderr, "Warning: config file not found, using only env vars and flags\n")
		} else if ok {
			return nil, fmt.Errorf("config file not found: %w", err)
		} else {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}
	return &cfg, nil
}
