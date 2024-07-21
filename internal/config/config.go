package config

import (
	"encoding/json"
	"github.com/caarlos0/env/v9"
	"github.com/pkg/errors"
	"os"
)

type Config struct {
	ServerAddr     string  `env:"SERVER_ADDR,required" json:"server_address"`
	ProductionMode bool    `env:"PRODUCTION_MODE" envDefault:"false" json:"production_mode"`
	SentryDsn      *string `env:"SENTRY_DSN" json:"sentry_dsn"`

	Discord struct {
		PublicKey     string   `env:"PUBLIC_KEY,required" json:"public_key"`
		AllowedGuilds []uint64 `env:"ALLOWED_GUILDS,required" json:"allowed_guilds"`
	} `envPrefix:"DISCORD_" json:"discord"`

	Patreon struct {
		ClientId       string `env:"CLIENT_ID,required" json:"client_id"`
		ClientSecret   string `env:"CLIENT_SECRET,required" json:"client_secret"`
		CampaignId     int    `env:"CAMPAIGN_ID,required" json:"campaign_id"`
		TokensFilePath string `env:"TOKENS_FILE_PATH" envDefault:"tokens.json" json:"tokens_file_path"`
	} `envPrefix:"PATREON_" json:"patreon"`

	Tiers map[uint64]string `env:"TIERS" json:"tiers"`
}

func LoadConfig() (Config, error) {
	var conf Config
	if _, err := os.Stat("config.json"); err == nil {
		f, err := os.Open("config.json")
		if err != nil {
			return Config{}, errors.Wrap(err, "failed to open config.json")
		}

		if err := json.NewDecoder(f).Decode(&conf); err != nil {
			return Config{}, errors.Wrap(err, "failed to decode config.json")
		}

		if conf.Patreon.TokensFilePath == "" {
			conf.Patreon.TokensFilePath = "tokens.json"
		}
	} else if errors.Is(err, os.ErrNotExist) { // If config.json does not exist, load from envvars
		if err := env.Parse(&conf); err != nil {
			return Config{}, errors.Wrap(err, "failed to parse env vars")
		}
	} else {
		return conf, errors.Wrap(err, "failed to check if config.json exists")
	}

	return conf, nil
}
