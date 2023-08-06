package config

import (
	"github.com/caarlos0/env/v9"
)

type Config struct {
	ServerAddr     string `env:"SERVER_ADDR,required"`
	ProductionMode bool   `env:"PRODUCTION_MODE" envDefault:"false"`
	SentryDsn      string `env:"SENTRY_DSN"`

	Discord struct {
		PublicKey     string   `env:"PUBLIC_KEY,required"`
		AllowedGuilds []uint64 `env:"ALLOWED_GUILDS,required"`
	} `envPrefix:"DISCORD_"`

	Patreon struct {
		ClientId       string `env:"CLIENT_ID,required"`
		ClientSecret   string `env:"CLIENT_SECRET,required"`
		CampaignId     int    `env:"CAMPAIGN_ID,required"`
		TokensFilePath string `env:"TOKENS_FILE_PATH" envDefault:"tokens.json"`
	} `envPrefix:"PATREON_"`
}

func LoadConfig() (conf Config, err error) {
	err = env.Parse(&conf)
	return
}
