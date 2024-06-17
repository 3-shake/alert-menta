package utils

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pkoukk/tiktoken-go"
	"github.com/spf13/viper"
)

// Root structure of information read from config file
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

type File struct {
	Path string
	Data string
}

func NewConfig(filename string) (*Config, error) {
	// Initialize a logger
	logger := log.New(
		os.Stdout, "[alert-menta utils] ",
		log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix,
	)

	// Get the directory and file name from variable filename
	dir, file := filepath.Split(filename)
	base, ext := filepath.Base(file)[:len(filepath.Base(file))-len(filepath.Ext(file))], filepath.Ext(file)[1:]

	// Read the config file
	viper.SetConfigName(base)
	viper.SetConfigType(ext)
	viper.AddConfigPath(dir)
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

func GetAllFiles(dir string) ([]File, error) {
	// Initialize a logger
	logger := log.New(
		os.Stdout, "[alert-menta utils] ",
		log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix,
	)

	var files []File

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if (info.IsDir() && info.Name() == "dist") || (info.IsDir() && info.Name() == ".git") || (info.IsDir() && info.Name() == ".github") {
			return filepath.SkipDir
		}
		if info.Name() == "go.mod" || info.Name() == "go.sum" || info.Name() == "alert-menta" {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		data, _ := getFileData(path)
		files = append(files, File{Path: path, Data: data})
		return nil
	})
	if err != nil {
		logger.Fatalf("Error reading files, %s", err)
		return nil, err
	}

	// count tokens
	result := 0
	encoding := "gpt-3.5-turbo"
	tke, err := tiktoken.EncodingForModel(encoding)
	if err != nil {
		logger.Fatalf("Error encoding for model, %s", err)
		return nil, err
	}
	logTokens := ""
	for _, file := range files {
		token := countTokens(tke, file.Data)
		result += token
		logTokens += fmt.Sprintf("tiktoken: %d, path: %s\n", token, file.Path)
	}
	logger.Println(logTokens+"all tokens:", result)

	return files, nil
}

func getFileData(path string) (string, error) {
	// Initialize a logger
	logger := log.New(
		os.Stdout, "[alert-menta utils] ",
		log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix,
	)

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		logger.Fatalf("Error opening file, %s", err)
		return "", err
	}
	defer file.Close()

	// Read the file
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		logger.Fatalf("Error reading file, %s", err)
		return "", err
	}

	// Print the file data
	// logger.Println("Data:", buf.String())
	return buf.String(), nil
}

func countTokens(tokenEncoder *tiktoken.Tiktoken, data string) int {
	// Reference from https://github.com/pkoukk/tiktoken-go
	token := tokenEncoder.Encode(data, nil, nil)
	return len(token)
}
