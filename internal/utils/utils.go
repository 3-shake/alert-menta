package utils

import (
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Root structure of information read from config file
type Config struct {
	System System `yaml:"system"`
	Ai     Ai     `yaml:"ai"`
}

type System struct {
	Debug SystemDebug `yaml:"debug"`
}

type SystemDebug struct {
	Log_level string `yaml:"log_level"`
}

type Ai struct {
	Commands map[string]Command `yaml:"commands"`
	Provider string             `yaml:"provider"`
	OpenAI   OpenAI             `yaml:"openai"`
	VertexAI VertexAI           `yaml:"vertexai"`
}

type Command struct {
	Description   string `yaml:"description"`
	System_prompt string `yaml:"system_prompt"`
}

type OpenAI struct {
	Model string `yaml:"model"`
}

type VertexAI struct {
	Model   string `yaml:"model"`
	Project string `yaml:"project"`
	Region  string `yaml:"region"`
}

func NewConfig(filename string) (*Config, error) {
	// Initialize a logger
	logger := log.New(
		os.Stdout, "[alert-menta utils] ",
		log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix,
	)

	// Get the directory and file name from variable filename
	dir, file := filepath.Split(filename)
	// Extract base part and extension part
	base, ext := filepath.Base(file)[:len(filepath.Base(file))-len(filepath.Ext(file))], filepath.Ext(file)[1:]

	// Read the config file
	viper.SetConfigName(base)
	viper.SetConfigType(ext)
	viper.AddConfigPath(dir)
	err := viper.ReadInConfig()
	if err != nil {
		logger.Fatalf("Error reading config file, %v", err)
	}

	// Unmarshal the config file
	cfg := new(Config)
	err = viper.Unmarshal(cfg)
	if err != nil {
		logger.Fatalf("Error unmarshal read config, %v", err)
		return nil, err
	}

	// Print the config
	logger.Println("Config:", cfg)
	return cfg, nil
}
