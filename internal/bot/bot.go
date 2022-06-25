package bot

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/vanadium23/wallabag-telegram-bot/internal/wallabag"
	"github.com/vanadium23/wallabag-telegram-bot/internal/worker"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
	"mvdan.cc/xurls"
)

func middlewareFilterUser(filterUsers []string) tele.MiddlewareFunc {
	allowedUsers := map[string]bool{}
	for _, s := range filterUsers {
		allowedUsers[s] = true
	}
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if _, ok := allowedUsers[c.Sender().Username]; !ok {
				return c.Send("You are not allowed to use bot")
			}
			return next(c)
		}
	}
}

func listenAckQueue(bot *tele.Bot, ackQueue chan worker.SaveURLRequest, ctx context.Context) {
	select {
	case ackMsg := <-ackQueue:
		msg := fmt.Sprintf("Article %s successfully saved to Wallabag", ackMsg.URL)
		bot.Send(tele.ChatID(ackMsg.ChatID), msg)
	case <-ctx.Done():
		return
	}
}

func StartTelegramBot(
	telegramBotToken string,
	pollInterval time.Duration,
	filterUsers []string,
	// for handlers
	wallabagClient wallabag.WallabagClient,
	worker worker.Worker,
	ackQueue chan worker.SaveURLRequest,
	ctx context.Context,
) *tele.Bot {
	pref := tele.Settings{
		Token:  telegramBotToken,
		Poller: &tele.LongPoller{Timeout: pollInterval},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	// use logger
	b.Use(middleware.Logger())
	b.Use(middlewareFilterUser(filterUsers))

	// handlers
	b.Handle("/start", func(c tele.Context) error {
		return c.Send("Welcome to wallabot. Just send me a string, and I will save it.")
	})
	b.Handle("/random", func(c tele.Context) error {
		articles, err := wallabagClient.FetchArticles(1, 30, 0)
		if err != nil {
			return c.Send("Wallabag failed with error: %v", err)
		}
		article := articles[rand.Intn(len(articles))]
		msg := fmt.Sprintf("I've found random article: %s", article.Url)
		return c.Send(msg)
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

		articles, err := wallabagClient.FetchArticles(1, count, 0)
		if err != nil {
			return c.Send("Wallabag failed with error: %v", err)
		}
		for i, article := range articles {
			msg := fmt.Sprintf("%d. %s", i+1, article.Url)
			c.Send(msg)
		}
		return nil
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
	return b
}
