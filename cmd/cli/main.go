package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	logrus "github.com/sirupsen/logrus"

	"github.com/vanadium23/wallabag-telegram-bot/internal/bot"
	"github.com/vanadium23/wallabag-telegram-bot/internal/tagging"
	"github.com/vanadium23/wallabag-telegram-bot/internal/wallabag"
)

type WallabagTelegramConfig struct {
	TelegramToken        string   `json:"token"`
	WallabagSite         string   `json:"wallabag_site"`
	WallabagClientID     string   `json:"client_id"`
	WallabagClientSecret string   `json:"client_secret"`
	WallabagUsername     string   `json:"username"`
	WallabagPassword     string   `json:"password"`
	WallabagDefaultTags  string   `json:"default_tags"`
	TelegramAllowedUsers []string `json:"filter_users"`
	OpenAISecretKey      string   `json:"open_ai_secret_key"`
	OpenAIProxyUrl       *url.URL `json:"open_ai_proxy_url"`
	OpenrouterApiKey     string   `json:"openrouter_api_key"`
	OpenrouterModel      string   `json:"openrouter_model"`
}

func readConfig() (WallabagTelegramConfig, error) {
	viper.SetConfigName("wallabag")
	viper.AddConfigPath("$HOME/.config/t.me")
	viper.AddConfigPath(".")
	viper.ReadInConfig()

	viper.SetConfigFile(".env")
	viper.MergeInConfig()

	viper.SetEnvPrefix("WALLABOT")
	viper.AutomaticEnv()

	c := WallabagTelegramConfig{}

	Token := viper.GetString("token")
	if Token == "" {
		return c, errors.New("token cannot be empty")
	}

	Site := viper.GetString("wallabag_site")
	if Site == "" {
		return c, errors.New("wallabag_site cannot be empty")
	}
	if !strings.HasPrefix(Site, "http") {
		Site = fmt.Sprintf("https://%s", Site)
	}

	ClientID := viper.GetString("client_id")
	if ClientID == "" {
		return c, errors.New("client_id cannot be empty")
	}

	ClientSecret := viper.GetString("client_secret")
	if ClientSecret == "" {
		return c, errors.New("client_secret cannot be empty")
	}

	Username := viper.GetString("username")
	if Username == "" {
		return c, errors.New("username cannot be empty")
	}

	Password := viper.GetString("password")
	if Password == "" {
		return c, errors.New("password cannot be empty")
	}

	FilterUsers := viper.GetStringSlice("filter_users")
	DefaultTags := viper.GetString("default_tags")
	OpenAISecretKey := viper.GetString("openai_secret_key")
	OpenrouterApiKey := viper.GetString("openrouter_api_key")
	OpenrouterModel := viper.GetString("openrouter_model")

	OpenAIProxyString := viper.GetString("openai_proxy_url")
	var OpenAIProxyUrl *url.URL
	if OpenAIProxyString != "" {
		proxyUrl, err := url.Parse(OpenAIProxyString)
		if err != nil {
			return c, errors.Join(errors.New("wrong proxy format"), err)
		}
		OpenAIProxyUrl = proxyUrl
	}

	return WallabagTelegramConfig{
		TelegramToken:        Token,
		WallabagSite:         Site,
		WallabagClientID:     ClientID,
		WallabagClientSecret: ClientSecret,
		WallabagUsername:     Username,
		WallabagPassword:     Password,
		WallabagDefaultTags:  DefaultTags,
		TelegramAllowedUsers: FilterUsers,
		OpenAISecretKey:      OpenAISecretKey,
		OpenAIProxyUrl:       OpenAIProxyUrl,
		OpenrouterApiKey:     OpenrouterApiKey,
		OpenrouterModel:      OpenrouterModel,
	}, nil
}

const timeOut = 60

func main() {
	log := logrus.New()
	log.Out = os.Stdout
	log.Info("Init")

	config, err := readConfig()
	if err != nil {
		log.Fatalf("Error found while reading config: %v", err)
	}

	wallabagClient := wallabag.NewWallabagClient(
		http.DefaultClient,
		config.WallabagSite,
		config.WallabagClientID,
		config.WallabagClientSecret,
		config.WallabagUsername,
		config.WallabagPassword,
		config.WallabagDefaultTags,
	)

	tagger := tagging.NewTagger(
		config.OpenAISecretKey,
		config.OpenAIProxyUrl,
		config.OpenrouterApiKey,
		config.OpenrouterModel,
	)
	b := bot.StartTelegramBot(
		config.TelegramToken,
		timeOut*time.Second,
		config.TelegramAllowedUsers,
		wallabagClient,
		tagger,
	)
	if b != nil {
		b.Start()
	}
}
