package inits

import (
	"github.com/spf13/viper"
)

func InitConfig() {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("json")   // or viper.SetConfigType("YAML")
	viper.AddConfigPath("../")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic("failed to read config file")
	}
}
