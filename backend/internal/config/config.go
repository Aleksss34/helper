package config

import (
	"flag"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type GatewayConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}
type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	DBName   string `yaml:"db_name"`
	Password string `yaml:"password"`
	Sslmode  string `yaml:"sslmode"`
}
type QdrantConfig struct {
	Host           string  `yaml:"host"`
	Port           int     `yaml:"port"`
	NameCollection string  `yaml:"name-collection"`
	BatchSize      int     `yaml:"batch-size"`
	LimitPoints    uint64  `yaml:"limit-points"`
	ScoreThreshold float32 `yaml:"score-threshold"`
}

type ParserConfig struct {
	BrowserPath  string `yaml:"browser-path"`
	CommandsPath string `yaml:"commands-path"`
}

type Config struct {
	Env string `yaml:"env"`

	TimeoutServer int64          `yaml:"timeout-server"`
	Gateway       GatewayConfig  `yaml:"gateway"`
	Postgres      PostgresConfig `yaml:"postgres"`
	Parser        ParserConfig   `yaml:"parser"`
	Qdrant        QdrantConfig   `yaml:"qdrant"`
	ApiKeyGroq    string         `env:"KEY_GROQ"`
}

func MustLoad() *Config {
	path := fetchPathConfig()
	if path == "" {
		panic("Couldn't find the path to the config")
	}
	return MustLoadPath(path)
}

func MustLoadPath(path string) *Config {
	var cfg Config
	_ = godotenv.Load()
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("Failed to unmarshal the config, error: " + err.Error())
	}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic("Failed to read env variables, error: " + err.Error())
	}
	return &cfg
}

func fetchPathConfig() string {
	var res string
	flag.StringVar(&res, "config", "", "config path")
	flag.Parse()
	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}
	return res
}
