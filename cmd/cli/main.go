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

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/mattn/go-sqlite3"
	logrus "github.com/sirupsen/logrus"
	xurls "mvdan.cc/xurls"

	"github.com/vanadium23/wallabag-telegram-bot/internal/articles"
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

	filterUsers := map[string]bool{}
	for _, s := range botInfo.FilterUsers {
		filterUsers[s] = true
	}

	bot, err := tgbotapi.NewBotAPI(botInfo.Token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false

	log.Infof("Authorized on account %s", bot.Self.UserName)

	go func() {
		for {
			select {
			case ackMsg := <-ackQueue:
				log.Info("Sending")
				msg := tgbotapi.NewMessage(ackMsg.ChatID, ackMsg.URL)

				bot.Send(msg)
			case <-ctx.Done():
				return
			}
		}
	}()

	rxStrict := xurls.Strict

	offset := 0

	for {
		u := tgbotapi.NewUpdate(offset)

		updates, err := bot.GetUpdates(u)

		if err != nil {
			log.Error(err)
		}

		timer := time.NewTimer(timeOut * time.Second)

		for _, update := range updates {
			offset = 1 + update.UpdateID

			if update.Message == nil { // ignore any non-Message Updates
				continue
			}

			log.Infof("Telegram received: %s", update.Message.Text)

			if _, ok := filterUsers[update.Message.From.UserName]; !ok {
				log.Infof("Telegram discards as it is from user: %s", update.Message.From.UserName)
				continue
			}

			for _, r := range rxStrict.FindAllString(update.Message.Text, -1) {
				log.Infof("Found URL: %s", r)
				worker.SendToDisk(r, update.Message.Chat.ID, 0)
			}
		}

		select {
		case <-signalCh:
			cancel()
			os.Exit(0)
		case <-timer.C:
			break
		}
	}
}
