package main

import (
	"fmt"
	"os"
	
	"github.com/spf13/viper"
)

func main() {
	// Try .cpctl.yaml first, fall back to birdy.yaml
	viper.SetConfigName(".cpctl")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	viper.AddConfigPath("../..")

	viper.SetEnvPrefix("CPCTL")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("playground.name", "birdy-playground")
	viper.SetDefault("playground.data_dir", "./data")
	viper.SetDefault("aws.region", "eu-central-1")
	viper.SetDefault("localstack.enabled", true)
	viper.SetDefault("localstack.endpoint", "http://localhost:4566")
	viper.SetDefault("localstack.port", 4566)
	viper.SetDefault("kind.enabled", true)
	viper.SetDefault("kind.cluster_name", "birdy-local")
	viper.SetDefault("kind.kubeconfig", os.ExpandEnv("$HOME/.kube/config"))

	// Load config (silent if not found)
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading .cpctl.yaml: %v\n", err)
		// Try fallback birdy.yaml
		viper.SetConfigName("birdy")
		if err := viper.ReadInConfig(); err != nil {
			fmt.Printf("Error reading birdy.yaml: %v\n", err)
			fmt.Println("Using default config (no .cpctl.yaml or birdy.yaml)")
		} else {
			fmt.Printf("Successfully loaded birdy.yaml from: %s\n", viper.ConfigFileUsed())
		}
	} else {
		fmt.Printf("Successfully loaded .cpctl.yaml from: %s\n", viper.ConfigFileUsed())
	}

	// Get values
	fmt.Printf("playground.data_dir: %s\n", viper.GetString("playground.data_dir"))
	fmt.Printf("playground.name: %s\n", viper.GetString("playground.name"))
	
	// Check if file exists
	if _, err := os.Stat("birdy.yaml"); err == nil {
		fmt.Println("birdy.yaml exists in current directory")
	} else {
		fmt.Printf("birdy.yaml does not exist in current directory: %v\n", err)
	}
	
	// Check current directory
	cwd, _ := os.Getwd()
	fmt.Printf("Current directory: %s\n", cwd)
	
	// Check if kind/ exists
	if _, err := os.Stat("kind"); err == nil {
		fmt.Println("kind/ directory exists")
	} else {
		fmt.Printf("kind/ directory does not exist: %v\n", err)
	}
}
