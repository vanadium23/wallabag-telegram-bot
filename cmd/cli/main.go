package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	logrus "github.com/sirupsen/logrus"

	"github.com/vanadium23/wallabag-telegram-bot/internal/bot"
	"github.com/vanadium23/wallabag-telegram-bot/internal/wallabag"
)

type WallabagTelegramConfig struct {
	TelegramToken        string   `json:"token"`
	WallabagSite         string   `json:"wallabag_site"`
	WallabagClientID     string   `json:"client_id"`
	WallabagClientSecret string   `json:"client_secret"`
	WallabagUsername     string   `json:"username"`
	WallabagPassword     string   `json:"password"`
	TelegramAllowedUsers []string `json:"filter_users"`
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

	return WallabagTelegramConfig{
		TelegramToken:        Token,
		WallabagSite:         Site,
		WallabagClientID:     ClientID,
		WallabagClientSecret: ClientSecret,
		WallabagUsername:     Username,
		WallabagPassword:     Password,
		TelegramAllowedUsers: FilterUsers,
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
	)
	b := bot.StartTelegramBot(
		config.TelegramToken,
		timeOut*time.Second,
		config.TelegramAllowedUsers,
		wallabagClient,
	)
	if b != nil {
		b.Start()
	}
}
