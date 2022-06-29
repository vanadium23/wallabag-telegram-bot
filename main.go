package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/Strubbl/wallabago/v7"
	"github.com/sirupsen/logrus"
	"github.com/vanadium23/wallabag-telegram-bot/internal/worker"
	tele "gopkg.in/telebot.v3"
	"mvdan.cc/xurls"
)

type Config struct {
	TelegramToken        string
	WallabagSite         string
	WallabagClientID     string
	WallabagClientSecret string
	WallabagUsername     string
	WallabagPassword     string
	FilterUsers          map[string]bool
}

func readConfig() (Config, error) {
	return Config{}, nil
}

const (
	archiveText   = "archive"
	unarchiveText = "unarchive"
)

func middlewareFilterUser(filterUsers map[string]bool) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if _, ok := filterUsers[c.Sender().Username]; !ok {
				return c.Send("You are not allowed to use bot")
			}
			return next(c)
		}
	}
}

// formCallbackQuery generates same string as InlineButton.CallbackUnique from telebot
func formCallbackQuery(text string) string {
	return "\f" + text
}

func formInlineButtons(entryID int, archive bool) *tele.ReplyMarkup {
	// buttons
	entry := strconv.Itoa(entryID)
	selector := &tele.ReplyMarkup{}
	btn := tele.Btn{}
	if archive {
		btn = selector.Data("✅", archiveText, entry)
	} else {
		btn = selector.Data("📥", unarchiveText, entry)
	}
	selector.Inline(
		selector.Row(btn),
	)
	return selector
}

func main() {
	// создать логирование
	log := logrus.New()
	log.Out = os.Stdout
	log.Info("Logger started")
	// получить, провалидировать и нормализовать конфиг
	config, err := readConfig()
	if err != nil {
		log.Fatalf("Error while reading config: %v", err)
	}
	// создать клиент wallabago
	wbConfig := wallabago.NewWallabagConfig(
		config.WallabagSite,
		config.WallabagClientID,
		config.WallabagClientSecret,
		config.WallabagUsername,
		config.WallabagPassword,
	)
	wallabago.SetConfig(wbConfig)
	// создать бота с handlers
	pref := tele.Settings{
		Token: config.TelegramToken,
		// TODO: move to config
		Poller: &tele.LongPoller{Timeout: 60 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatalf("Error while reading creating telegram bot: %v", err)
	}

	// use logger
	b.Use(middlewareFilterUser(config.FilterUsers))

	// handlers
	b.Handle("/start", func(c tele.Context) error {
		return c.Send("Welcome to wallabot. Just send me a string, and I will save it.")
	})
	b.Handle("/random", func(c tele.Context) error {
		result, err := wallabago.GetEntries(wallabago.APICall, -1, -1, "", "", 1, 30, "")
		if err != nil {
			return c.Send("Wallabag failed with error: %v", err)
		}
		articles := result.Embedded.Items
		article := articles[rand.Intn(len(articles))]
		msg := fmt.Sprintf("I've found random article: %s", article.URL)
		return c.Send(msg, formInlineButtons(article.ID, true))
	})
	b.Handle("/recent", func(c tele.Context) error {
		count := 1
		args := c.Args()
		for _, arg := range args {
			argCount, err := strconv.ParseInt(arg, 0, 64)
			if err == nil {
				count = int(argCount)
			}
		}

		result, err := wallabago.GetEntries(wallabago.APICall, -1, -1, "", "", 1, count, "")
		if err != nil {
			return c.Send("Wallabag failed with error: %v", err)
		}
		articles := result.Embedded.Items
		for i, article := range articles {
			msg := fmt.Sprintf("%d. %s", i+1, article.URL)
			c.Send(msg, formInlineButtons(article.ID, true))
		}
		return nil
	})
	b.Handle(formCallbackQuery(archiveText), func(c tele.Context) error {
		entryID, err := strconv.ParseInt(c.Callback().Data, 10, 64)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during archiving entry: %v", err),
			})
		}
		wallabago.
			err = wallabagClient.UpdateArticle(int(entryID), 1)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during archiving entry: %v", err),
			})
		}
		c.Bot().EditReplyMarkup(c.Update().Callback.Message, formInlineButtons(int(entryID), false))
		return c.Respond(&tele.CallbackResponse{
			CallbackID: c.Callback().ID,
			Text:       "Entry was successfully archived",
		})
	})
	b.Handle(formCallbackQuery(unarchiveText), func(c tele.Context) error {
		entryID, err := strconv.ParseInt(c.Callback().Data, 10, 64)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during restoring entry: %v", err),
			})
		}
		err = wallabagClient.UpdateArticle(int(entryID), 0)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during restoring entry: %v", err),
			})
		}
		c.Bot().EditReplyMarkup(c.Update().Callback.Message, formInlineButtons(int(entryID), true))
		return c.Respond(&tele.CallbackResponse{
			CallbackID: c.Callback().ID,
			Text:       "Entry was successfully saved back.",
		})
	})
	b.Handle(tele.OnText, func(c tele.Context) error {
		for _, r := range xurls.Strict.FindAllString(c.Message().Text, -1) {
			// TODO: fix messageID
			worker.SendToDisk(r, c.Chat().ID, 0)
		}
		return c.Send("Received article, now saving")
	})

	// start bot
	go listenAckQueue(b, ackQueue, ctx)
	go b.Start()
}
