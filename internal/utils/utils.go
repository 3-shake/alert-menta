package utils

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

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
	LogLevel string `yaml:"log_level" mapstructure:"log_level"`
}

type Ai struct {
	Commands map[string]Command `yaml:"commands"`
	Provider string             `yaml:"provider"`
	OpenAI   OpenAI             `yaml:"openai"`
	VertexAI VertexAI           `yaml:"vertexai"`
}

type Command struct {
	Description   string `yaml:"description"`
	SystemPrompt  string `yaml:"system_prompt" mapstructure:"system_prompt"`
	RequireIntent bool   `yaml:"require_intent" mapstructure:"require_intent"`
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
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Unmarshal the config file
	cfg := new(Config)
	err = viper.Unmarshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Print the config
	logger.Println("Config:", cfg)
	return cfg, nil
}

func DownloadImage(url string, token string) ([]byte, string, error) {
	// Create a new HTTP client
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to create a new request: %w", err)
	}

	// Download the image with the token
	req.Header.Set("Authorization", "Bearer "+token) // set token to header
	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to get a response: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Write the response body to the temporary file
	file, err := os.CreateTemp("", "downloaded-image-*")
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to create a temporary file: %w", err)
	}
	defer func() {
		log.Println("remove", file.Name(), "Content-Type:", resp.Header.Get("Content-Type"))
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to write the response body to the temporary file: %w", err)
	}

	// Read image data from the temporary file
	data, err := os.ReadFile(file.Name())
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to read the file: %w", err)
	}

	// Get the extension of the image
	contentType := resp.Header.Get("Content-Type")
	imageRegex := regexp.MustCompile(`.+/(.*)`)
	matches := imageRegex.FindAllStringSubmatch(contentType, -1)
	if len(matches) == 0 {
		return []byte{}, "", fmt.Errorf("failed to get the extension of the image")
	}
	ext := matches[0][1]

	return data, ext, nil
}

func ImageToBase64(data []byte, ext string) string {
	base64img := base64.StdEncoding.EncodeToString(data)
	return "data:image/" + ext + ";base64," + base64img
}

func ExtractImageURLs(body string) []string {
	imageRegex := regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)
	matches := imageRegex.FindAllStringSubmatch(body, -1)
	var urls []string
	for _, match := range matches {
		urls = append(urls, match[2])
	}
	return urls
}
