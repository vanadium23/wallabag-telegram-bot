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

	"github.com/vanadium23/wallabag-telegram-bot/internal/wallabag"
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

type saveURLRequest struct {
	URL       string
	ChatID    int64
	MessageID int
}

const rescanInterval = 3600

func sqlite3Handler(diskQueue, reqQueue, diskAckQueue, ackQueue chan saveURLRequest) {
	var database *sql.DB
	var statement *sql.Stmt
	var err error
	database, err = sql.Open("sqlite3", "./wallabag.db")

	if err != nil {
		log.Fatalf("%v", err)
	}

	if statement, err = database.Prepare(`CREATE TABLE IF NOT EXISTS Requests (	
			id INTEGER PRIMARY KEY AUTOINCREMENT, 
			URL TEXT, 
			ChatID INTEGER,
			MessageID INTEGER,
			saved INTEGER)`); err != nil {
		log.Fatalf("%v", err)
	}

	if _, err = statement.Exec(); err != nil {
		log.Fatalf("%v", err)
	}

	if statement, err = database.Prepare(`CREATE INDEX IF NOT EXISTS URLIndex ON Requests(URL);`); err != nil {
		log.Fatalf("%v", err)
	}

	if _, err = statement.Exec(); err != nil {
		log.Fatalf("%v", err)
	}

	go func() {

		for {
			timer := time.NewTimer(rescanInterval * time.Second)

			var rows *sql.Rows
			if rows, err = database.Query(`
		SELECT URL, ChatID, MessageID
		FROM Requests
		WHERE saved == 0
		`); err != nil {
				log.Fatalf("%v", err)
			}

			var URL string
			var ChatID int64
			var MessageID int

			for rows.Next() {
				if err := rows.Scan(&URL, &ChatID, &MessageID); err != nil {
					log.Error("Cannot read url from database")
					continue
				}
				log.Infof("Unfinished %s, %d, %d", URL, ChatID, MessageID)
				reqQueue <- saveURLRequest{URL: URL, ChatID: ChatID, MessageID: MessageID}
			}

			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
		}
	}()

	go func() {
		for {
			var r saveURLRequest
			select {
			case r = <-diskQueue:
			case <-ctx.Done():
				return
			}

			var count int
			row := database.QueryRow("SELECT COUNT(*) FROM Requests WHERE URL = ?", r.URL)
			err := row.Scan(&count)

			if err != nil {
				log.Errorf("Fail to get count of URL: %v", err)
				continue
			}

			if count != 0 {
				log.Infof("Skip existing URL: %s", r.URL)
				continue
			}

			log.Infof("Saving request to disk first: %s, %d, %d", r.URL, r.ChatID, r.MessageID)

			statement, err := database.Prepare("INSERT INTO Requests (URL, ChatID, MessageID, saved) VALUES (?, ?, ?, 0)")
			if err != nil {
				log.Errorf("Fail to insert request to SQLite: %v", err)
				continue
			}
			_, err = statement.Exec(r.URL, r.ChatID, r.MessageID)
			if err != nil {
				log.Errorf("Fail to insert request to SQLite: %v", err)
				continue
			}
			reqQueue <- r
		}
	}()

	go func() {
		for {
			var r saveURLRequest
			select {
			case r = <-diskAckQueue:
			case <-ctx.Done():
				return
			}
			log.Infof("Update URL as saved: %s, %d, %d", r.URL, r.ChatID, r.MessageID)

			var count int
			row := database.QueryRow("SELECT COUNT(*) FROM Requests WHERE URL = ?", r.URL)
			err := row.Scan(&count)

			if err != nil {
				log.Errorf("Fail to get count of URL: %v", err)
				continue
			}

			if count == 0 {
				log.Errorf("This URL should exist: %s", r.URL)
			}

			statement, err := database.Prepare("UPDATE Requests SET saved = 1 WHERE URL = ?")
			if err != nil {
				log.Errorf("Fail to update request to SQLite: %v", err)
				continue
			}
			_, err = statement.Exec(r.URL)
			if err != nil {
				log.Errorf("Fail to update request to SQLite: %v", err)
				continue
			}
			ackQueue <- r
		}
	}()
}

func wallabagHandler(wc wallabag.WallabagClient, reqQueue, diskAckQueue chan saveURLRequest) {
	go func() {
		for {
			var r saveURLRequest
			select {
			case r = <-reqQueue:
			case <-ctx.Done():
				return
			}
			_, err := wc.CreateArticle(r.URL)
			if err != nil {
				log.Error(err)
				continue
			}

			log.Infof("Wallabag says it is saved: %s, %d, %d", r.URL, r.ChatID, r.MessageID)
			diskAckQueue <- r
		}

	}()
}

const timeOut = 60

func main() {
	diskQueue := make(chan saveURLRequest, 100)
	reqQueue := make(chan saveURLRequest, 100)
	diskAckQueue := make(chan saveURLRequest, 100)
	ackQueue := make(chan saveURLRequest, 100)
	wallabagClient := wallabag.NewWallabagClient(
		http.DefaultClient,
		fmt.Sprintf("https://%s", botInfo.Site),
		botInfo.ClientID,
		botInfo.ClientSecret,
		botInfo.Username,
		botInfo.Password,
	)

	wallabagHandler(wallabagClient, reqQueue, diskAckQueue)
	sqlite3Handler(diskQueue, reqQueue, diskAckQueue, ackQueue)

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
				diskQueue <- saveURLRequest{URL: r, ChatID: update.Message.Chat.ID}
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
