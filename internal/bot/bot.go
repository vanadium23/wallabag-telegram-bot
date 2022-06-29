package bot

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/vanadium23/wallabag-telegram-bot/internal/wallabag"
	tele "gopkg.in/telebot.v3"
	"mvdan.cc/xurls"
)

const (
	archiveText   = "archive"
	unarchiveText = "unarchive"
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

func StartTelegramBot(
	telegramBotToken string,
	pollInterval time.Duration,
	filterUsers []string,
	// for handlers
	wallabagClient wallabag.WallabagClient,
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

		articles, err := wallabagClient.FetchArticles(1, count, 0)
		if err != nil {
			return c.Send("Wallabag failed with error: %v", err)
		}
		for i, article := range articles {
			msg := fmt.Sprintf("%d. %s", i+1, article.Url)
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
		c.Send("Received message, finding articles and try to save")
		for _, r := range xurls.Strict.FindAllString(c.Message().Text, -1) {
			// TODO: fix messageID
			entry, err := wallabagClient.CreateArticle(r)
			if err != nil {
				c.Send(fmt.Sprintf("Found article %s, but save failed with err: %v", r, err))
				continue
			}
			c.Send(fmt.Sprintf("Found article %s and successfully saved with id: %d", entry.Url, entry.ID))
		}
		return nil
	})

	// start bot
	return b
}
