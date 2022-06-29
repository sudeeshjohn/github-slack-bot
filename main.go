package main

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("Failed to start the Bot")
	}
}

func run() error {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel) // Uncomment this line for info mode
	//zerolog.SetGlobalLevel(zerolog.DebugLevel) // Uncomment this line for debug mode
	log.Info().Msg("bot stats")
	/*err := godotenv.Load("./.env")
	if err != nil {
		log.Fatal().Msg("Error loading .env file")
	}*/
	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if len(botToken) == 0 {
		return fmt.Errorf("the environment variable SLACK_BOT_TOKEN must be set")
	}
	appToken := os.Getenv("SLACK_APP_TOKEN")
	if len(appToken) == 0 {
		return fmt.Errorf("the environment variable SLACK_APP_TOKEN must be set")
	}
	authToken := os.Getenv("GITHUB_OAUTH_TOKEN")
	if len(authToken) == 0 {
		return fmt.Errorf("the environment GITHUB_OAUTH_TOKEN must be set")
	}
	githubOrg := os.Getenv("GITHUB_ORG")
	if len(githubOrg) == 0 {
		return fmt.Errorf("the environment variable GITHUB_ORG must be set")
	}
	githubRepo := os.Getenv("GITHUB_REPO")
	if len(githubRepo) == 0 {
		return fmt.Errorf("the environment variable GITHUB_REPO must be set")
	}

	bot := NewBot(botToken)
	for {
		if err := bot.Start(); err != nil {
			return err
		}
		time.Sleep(5 * time.Second)
	}
}
