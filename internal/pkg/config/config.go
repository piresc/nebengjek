package config

import (
	"log"

	"github.com/piresc/nebengjek/internal/pkg/models"

	"github.com/spf13/viper"
)

func InitConfig(appName string) *models.Config {
	return readConfigs(appName)
}

func readConfigs(appName string) *models.Config {
	configs := &models.Config{}
	configTypes := []string{"main"}
	for _, configType := range configTypes {
		viper.SetConfigName(appName + "." + configType)
		viper.SetConfigType("yaml")
		viper.AddConfigPath("files") // Use files folder

		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				log.Fatal("config file not found")
			} else {
				log.Fatal("cannot read config file")
			}
		}
	}

	err := viper.Unmarshal(configs)
	if err != nil {
		log.Fatal("cannot unmarshal viper config")
	}

	return configs
}
