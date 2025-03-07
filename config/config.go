// DO NOT EDIT THIS FILE!!!!!!!
// This is not the config file. The config file can be found in the root directory as config.yaml
// (you may need to create it, copy config.sample.yml )

// This file is where the config file is parsed & loaded. Modifying this is highly discouraged (unless your a developer).

package config

import (
	"github.com/spf13/viper"
	"fmt"
)

type Config struct {
    // Version of the server
    Version string `mapstructure:"VERSION"`
	// Accessible server address
	CdnURL string `mapstructure:"CDN_URL"`
	// The port to run the server on
	ServerPort int `mapstructure:"SERVER_PORT"`
	// This enables extra logging, INCLUDING SENSETIVE INFORMATION like BLUESKY TOKENS. Useful for debugging with tools like insomnia. DO NOT USE ON PUBLIC SERVERS
	// Also requires all passwords start with "dev_" to work
	DeveloperMode bool `mapstructure:"DEVELOPER_MODE"`
    // Collects analytics on users.
    TrackAnalytics bool `mapstructure:"TRACK_ANALYTICS"`
    // Database type (mysql, postgres, sqlite)
    DatabaseType string `mapstructure:"DATABASE_TYPE"`
    // Database path
    DatabasePath string `mapstructure:"DATABASE_PATH"`

    UseXForwardedFor bool `mapstructure:"USE_X_FORWARDED_FOR"`

    ImgDisplayText string `mapstructure:"IMG_DISPLAY_TEXT"`
    ImgURLText string `mapstructure:"IMG_URL_TEXT"`
    VidDisplayText string `mapstructure:"VID_DISPLAY_TEXT"`
    VidURLText string `mapstructure:"VID_URL_TEXT"`
}

// Loads our config files.
func LoadConfig() (*Config, error) {
    viper.SetConfigName("config") // Name of the config file (without extension)
    viper.SetConfigType("yaml")  // File type
    viper.AddConfigPath(".")     // Look for config in the current directory
    viper.AddConfigPath("/config/") // Path for Docker setups
    
    // Read environment variables with a specific prefix
    viper.SetEnvPrefix("TWITTER_BRIDGE")

    // Set default values
    viper.SetDefault("VERSION", "1.0.4-beta") // wait till i forget to update this
    viper.SetDefault("SERVER_PORT", "3000")
    viper.SetDefault("DEVELOPER_MODE", false)
    viper.SetDefault("DATABASE_TYPE", "sqlite")
    viper.SetDefault("DATABASE_PATH", "./db/twitterbridge.db")
    viper.SetDefault("TRACK_ANALYTICS", true)
    viper.SetDefault("CDN_URL", "http://127.0.0.1:3000")
    viper.SetDefault("USE_X_FORWARDED_FOR", false)
    viper.SetDefault("IMG_DISPLAY_TEXT", "pic.twitter.com/{shortblob}")
    viper.SetDefault("VID_DISPLAY_TEXT", "pic.twitter.com/{shortblob}")
    viper.SetDefault("IMG_URL_TEXT", "http://127.0.0.1:3000/img/{shortblob}")
    viper.SetDefault("VID_URL_TEXT", "http://127.0.0.1:3000/img/{shortblob}")
    // Read config file
    if err := viper.ReadInConfig(); err != nil {
        fmt.Println("No config file found, relying on environment variables")
    }

    // Bind config to struct
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }

    return &config, nil
}