package config

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	MadisonAPIKey string
	MadisonURL    string
	Dms           string
	Port          string
}

var cfgInstance *Config

func LoadConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if _, err := os.Stat(configPath); err == nil {
		err = viper.ReadInConfig()
		if err != nil {
			return err
		}
	}

	viper.SetDefault("MADISON_API_KEY", "")
	viper.SetDefault("MADISON_URL", "https://madison.flant.com/api/events/custom/%s")
	viper.SetDefault("DMS", "AlertReceiver")
	viper.SetDefault("PORT", "80")

	cfgInstance = &Config{
		MadisonAPIKey: viper.GetString("MADISON_API_KEY"),
		MadisonURL:    viper.GetString("MADISON_URL"),
		Dms:           viper.GetString("DMS"),
		Port:          viper.GetString("PORT"),
	}

	if cfgInstance.MadisonAPIKey == "" {
		log.Fatal("missing required environment variable: MADISON_API_KEY")
	}

	return nil
}

func GetConfig() *Config {
	if cfgInstance == nil {
		log.Fatal("Config has not been initialized. Call LoadConfig first.")
	}
	return cfgInstance
}
