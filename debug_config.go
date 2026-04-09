package main

import (
	"fmt"
	"os"
	"strings"
	
	"github.com/spf13/viper"
)

type Config struct {
	Playground struct {
		Name    string
		DataDir string
	}
}

func main() {
	// Set up viper exactly like in the real code
	viper.SetConfigName("birdy")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	
	viper.SetEnvPrefix("CPCTL")
	viper.AutomaticEnv()
	
	// Set defaults
	viper.SetDefault("playground.name", "birdy-playground")
	viper.SetDefault("playground.data_dir", "./data")
	
	// Load config
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Config file used: %s\n", viper.ConfigFileUsed())
	fmt.Printf("playground.data_dir from viper: %s\n", viper.GetString("playground.data_dir"))
	fmt.Printf("playground.name from viper: %s\n", viper.GetString("playground.name"))
	
	// Try to unmarshal
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Printf("Error unmarshaling: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Config.Playground.DataDir: %s\n", cfg.Playground.DataDir)
	fmt.Printf("Config.Playground.Name: %s\n", cfg.Playground.Name)
	
	// Check if empty
	if strings.TrimSpace(cfg.Playground.DataDir) == "" {
		fmt.Println("ERROR: DataDir is empty after unmarshaling!")
	} else {
		fmt.Println("SUCCESS: DataDir has value")
	}
}
