package config

type Config struct {
	Port           string `yaml:"port"`
	Path           string `yaml:"path"`
	Db             string `yaml:"db"`
	Steamcmd       string `yaml:"steamcmd"`
	SteamAPIKey    string `yaml:"steamAPIKey"`
	OPENAI_API_KEY string `yaml:"OPENAI_API_KEY"`
	Prompt         string `yaml:"prompt"`
	Flag           string `yaml:"flag"`
}