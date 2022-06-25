package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"

	_ "github.com/mattn/go-sqlite3"
	logrus "github.com/sirupsen/logrus"

	"github.com/vanadium23/wallabag-telegram-bot/internal/articles"
	"github.com/vanadium23/wallabag-telegram-bot/internal/bot"
	"github.com/vanadium23/wallabag-telegram-bot/internal/wallabag"
	"github.com/vanadium23/wallabag-telegram-bot/internal/worker"
)

var log *logrus.Logger
var signalCh chan os.Signal
var ctx context.Context
var cancel context.CancelFunc

func init() {
	log = logrus.New()
	log.Out = os.Stdout

	botInfo.readConfig()

	log.Info("Init")

	signalCh = make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM)

	ctx, cancel = context.WithCancel(context.Background())
}

type BotInfo struct {
	Token        string   `json:"token"`
	Site         string   `json:"wallabag_site"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	FilterUsers  []string `json:"filter_users"`
}

var botInfo BotInfo

func (b *BotInfo) readConfig() {
	viper.SetConfigName("wallabag")
	viper.AddConfigPath("$HOME/.config/t.me")
	viper.AddConfigPath(".")
	viper.ReadInConfig()

	viper.SetConfigFile(".env")
	viper.MergeInConfig()

	viper.SetEnvPrefix("WALLABOT")
	viper.AutomaticEnv()

	b.Token = viper.GetString("token")
	b.Site = viper.GetString("wallabag_site")
	b.ClientID = viper.GetString("client_id")
	b.ClientSecret = viper.GetString("client_secret")
	b.Username = viper.GetString("username")
	b.Password = viper.GetString("password")
	b.FilterUsers = viper.GetStringSlice("filter_users")

	if b.Token == "" || b.Site == "" || b.ClientID == "" || b.ClientSecret == "" || b.Username == "" || b.Password == "" {
		log.Fatalf("Fail to parse bot token and wallabag credentials")
	}
}

const rescanInterval = 3600
const timeOut = 60

func main() {
	ackQueue := make(chan worker.SaveURLRequest, 100)
	wallabagClient := wallabag.NewWallabagClient(
		http.DefaultClient,
		fmt.Sprintf("https://%s", botInfo.Site),
		botInfo.ClientID,
		botInfo.ClientSecret,
		botInfo.Username,
		botInfo.Password,
	)
	var database *sql.DB
	var err error
	database, err = sql.Open("sqlite3", "./wallabag.db")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	articleRepo, err := articles.NewArticleRepo(database)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	worker := worker.NewWorker(wallabagClient, articleRepo, rescanInterval*time.Second)
	worker.Start(ctx, ackQueue)

	b := bot.StartTelegramBot(
		botInfo.Token,
		timeOut*time.Second,
		botInfo.FilterUsers,
		wallabagClient,
		worker,
		ackQueue,
		ctx,
	)

	select {
	case <-signalCh:
		b.Stop()
		cancel()
		os.Exit(0)
	}
}
