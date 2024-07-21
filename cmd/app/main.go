package main

import (
	"context"
	"encoding/json"
	"github.com/TicketsBot/subscriptions-app/internal/config"
	"github.com/TicketsBot/subscriptions-app/internal/server"
	"github.com/TicketsBot/subscriptions-app/pkg/patreon"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

func main() {
	conf, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	var logger *zap.Logger
	if conf.ProductionMode {
		if conf.SentryDsn != nil {
			if err := sentry.Init(sentry.ClientOptions{
				Dsn: *conf.SentryDsn,
			}); err != nil {
				panic(err)
			}

			defer sentry.Flush(time.Second * 2)

			logger, err = zap.NewProduction(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
				return zapcore.RegisterHooks(core, func(entry zapcore.Entry) error {
					if entry.Level == zapcore.ErrorLevel {
						hostname, _ := os.Hostname()

						sentry.CaptureEvent(&sentry.Event{
							Extra: map[string]any{
								"caller": entry.Caller.String(),
								"stack":  entry.Stack,
							},
							Level:      sentry.LevelError,
							Message:    entry.Message,
							ServerName: hostname,
							Timestamp:  entry.Time,
							Logger:     entry.LoggerName,
						})
					}

					return nil
				})
			}))
		} else {
			logger, err = zap.NewProduction()
		}
	} else {
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		panic(err)
	}

	tokens, err := readTokens(conf.Patreon.TokensFilePath)
	if err != nil {
		panic(err)
	}

	patreonClient := patreon.NewClient(
		conf,
		logger.With(zap.String("component", "patreon_client")),
		tokens,
	)

	pledgeCh := make(chan map[string]patreon.Patron)
	go startPatreonLoop(context.Background(), conf, logger, patreonClient, pledgeCh)

	server := server.NewServer(conf, logger.With(zap.String("component", "server")))

	go func() {
		for pledges := range pledgeCh {
			server.UpdatePledges(pledges)
		}
	}()

	if err := server.Run(); err != nil {
		panic(err)
	}
}

func startPatreonLoop(
	ctx context.Context,
	config config.Config,
	logger *zap.Logger,
	patreonClient *patreon.Client,
	ch chan map[string]patreon.Patron,
) {
	for {
		fetchPledges(ctx, config, logger, patreonClient, ch)
		time.Sleep(time.Minute)
	}
}

func fetchPledges(
	ctx context.Context,
	config config.Config,
	logger *zap.Logger,
	patreonClient *patreon.Client,
	ch chan map[string]patreon.Patron,
) {
	if patreonClient.Tokens.ExpiresAt.Before(time.Now()) {
		logger.Fatal(
			"Refresh token has already expired (expired at %s)",
			zap.Time("expires_at", patreonClient.Tokens.ExpiresAt),
		)
		return
	}

	if time.Until(patreonClient.Tokens.ExpiresAt) < time.Hour*24*3 {
		logger.Info(
			"Token expires in less than 3 days, refreshing",
			zap.Time("expires_at", patreonClient.Tokens.ExpiresAt),
		)

		ctx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()

		tokens, err := patreonClient.DoRefresh(ctx)
		if err != nil { // We can still continue if this fails
			logger.Error("Failed to refresh token", zap.Error(err))
		} else {
			logger.Info("Tokens refreshed successfully", zap.Time("expires_at", tokens.ExpiresAt))
			if err := writeTokens(config.Patreon.TokensFilePath, tokens); err != nil {
				logger.Error("Failed to write tokens to disk", zap.Error(err))
			} else {
				logger.Info("Tokens written to disk")
			}
		}

		cancel()
	}

	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()

	pledges, err := patreonClient.FetchPledges(ctx)
	if err != nil {
		logger.Error("Failed to fetch pledges", zap.Error(err))
		return
	}

	ch <- pledges
}

func writeTokens(path string, tokens patreon.Tokens) error {
	marshalled, err := json.Marshal(tokens)
	if err != nil {
		return err
	}

	return os.WriteFile(path, marshalled, 0600)
}

func readTokens(path string) (patreon.Tokens, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return patreon.Tokens{}, err
	}

	var tokens patreon.Tokens
	if err := json.Unmarshal(data, &tokens); err != nil {
		return patreon.Tokens{}, err
	}

	if tokens.ExpiresAt.IsZero() {
		tokens.ExpiresAt = time.Now().Add(time.Hour) // Attempt a refresh, likely this is the first run
	}

	return tokens, nil
}
