package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// LoadConfig loads configuration from YAML, JSON files and environment variables.
func LoadConfig() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	viper.SetConfigName("config") // config file name without extension
	viper.SetConfigType("yml")    // config file type
	viper.AddConfigPath(".")      // search config in the working directory
	viper.AutomaticEnv()          // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read the base configuration file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; can be ignored
			fmt.Println("Base config file (config.yml) not found, using environment variables and defaults.")
		} else {
			// Config file was found but another error was produced
			panic(fmt.Errorf("fatal error in base config file: %w", err))
		}

		// Merge the thread configuration file
		viper.SetConfigName("thread_config") // set the name of the json config file
		viper.SetConfigType("json")          // set the type of the config file
		viper.AddConfigPath("./config")      // path to look for the config file in

		if err := viper.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				fmt.Println("Thread config file (config/thread_config.json) not found.")
			} else {
				panic(fmt.Errorf("fatal error in thread config file: %w", err))
			}
		}
	}

	viper.SetConfigName("scanning_config") // set the name of the json config file
	viper.SetConfigType("json")            // set the type of the config file
	viper.AddConfigPath("./config")        // path to look for the config file in

	if err := viper.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Scanning config file (config/scanning_config.json) not found.")
		} else {
			panic(fmt.Errorf("fatal error in scanning config file: %w", err))
		}
	}
}
