package config

import (
	"time"

	"github.com/caarlos0/env/v8"
)

type ChatGPT struct {
	ApiKey              string        `env:"GPT_API_KEY,required"`
	InstructionFilePath string        `env:"GPT_INSTRUCTION_FILE_PATH"`
	InstructionText     string        `env:"GPT_INSTRUCTION_TEXT"`
	ApiTimeoutSecond    time.Duration `env:"GPT_API_TIMEOUT_SECOND"`
	ApiUrl              string        `env:"GPT_API_URL" envDefault:"https://api.openai.com/v1/chat/completions"`
	Model               string        `env:"GPT_MODEL" envDefault:"gpt-3.5-turbo"`
}

type Server struct {
	Port int `env:"PORT" envDefault:"8080"` // Heroku sets the PORT value automatically.
}

type Config struct {
	ChatGPT
	Server
}

func LoadConfigOrPanic() Config {
	var config *Config = new(Config)
	if err := env.Parse(config); err != nil {
		panic(err)
	}

	config.normalize()
	return *config
}

func (c *Config) normalize() {

	if c.ApiTimeoutSecond == 0 {
		c.ApiTimeoutSecond = time.Second * 30
	}

	if len(c.InstructionFilePath)+len(c.InstructionText) == 0 {
		panic("Either GPT_INSTRUCTION_FILE_PATH or GPT_INSTRUCTION_TEXT must be provided in the env")
	}
}
