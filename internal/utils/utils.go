package utils

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	System System `yaml:"system"`
	Ai     Ai     `yaml:"ai"`
	Github Github `yaml:"github"`
}

type Test struct {
	Mode bool   `yaml:"mode"`
	Name string `yaml:"name"`
}

type System struct {
	Debug SystemDebug `yaml:"debug"`
}

type SystemDebug struct {
	Mode      bool   `yaml:"mode"`
	Log_level string `yaml:"log_level"`
}

type Ai struct {
	Model    string             `yaml:"model"`
	Commands map[string]Command `yaml:"commands"`
}

type Command struct {
	// Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	System_prompt string `yaml:"system_prompt"`
}

type Github struct {
	Owner string `yaml:"owner"`
	Repo  string `yaml:"repo"`
}

func NewConfig() (*Config, error) {
	// Initialize a logger
	logger := log.New(
		os.Stdout, "[alert-menta utils] ",
		log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix,
	)

	// Read the config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./internal/config")
	err := viper.ReadInConfig()
	if err != nil {
		logger.Fatalf("Error reading config file, %s", err)
	}

	// Unmarshal the config file
	cfg := new(Config)
	err = viper.Unmarshal(cfg)
	if err != nil {
		logger.Fatalf("Error unmarshal read config, %s", err)
		return nil, err
	}

	// Print the config
	logger.Println("Config:", cfg)
	return cfg, nil
}
