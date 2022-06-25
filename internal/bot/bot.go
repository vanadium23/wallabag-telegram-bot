package bot

import (
	"context"
	"fmt"
	"log"
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
		msg := fmt.Sprintf("Article %s succesfully saved to Wallabag", ackMsg.URL)
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
