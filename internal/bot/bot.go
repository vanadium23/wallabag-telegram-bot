package bot

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/vanadium23/wallabag-telegram-bot/internal/tagging"
	"github.com/vanadium23/wallabag-telegram-bot/internal/wallabag"
	tele "gopkg.in/telebot.v3"
	"mvdan.cc/xurls"
)

const (
	archiveText   = "archive"
	unarchiveText = "unarchive"
	scrolledText  = "scrolled"
	rateText      = "rate"
)

const entryMessageTemplates = `
%s.

%s

📅 %s ⏳ %d min
`

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

func formatArticleMessage(message string, entry wallabag.WallabagEntry) string {
	return fmt.Sprintf(entryMessageTemplates,
		message,
		entry.Url,
		entry.CreatedAt.Format("2006-01-02"),
		entry.ReadingTime,
	)
}

func StartTelegramBot(
	telegramBotToken string,
	pollInterval time.Duration,
	filterUsers []string,
	// for handlers
	wallabagClient wallabag.WallabagClient,
	tagger tagging.Tagger,
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
		message := "I've found random article"
		return c.Send(formatArticleMessage(message, article), formInlineButtons(article.ID, true))
	})
	b.Handle("/recent", func(c tele.Context) error {
		count := 5
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
			message := fmt.Sprintf("Recent article №%d", i+1)
			c.Send(formatArticleMessage(message, article), formInlineButtons(article.ID, true))
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
	b.Handle(formCallbackQuery(scrolledText), func(c tele.Context) error {
		parts := strings.Split(c.Callback().Data, "|")
		if len(parts) < 2 {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during restoring entry: wrong callback data"),
			})
		}
		entryID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during restoring entry: %v", err),
			})
		}
		err = wallabagClient.AddTagsToArticle(int(entryID), []string{"scrolled"})
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during restoring entry: %v", err),
			})
		}
		c.Bot().EditReplyMarkup(c.Update().Callback.Message, formInlineButtons(int(entryID), parts[1] == "0"))
		return c.Respond(&tele.CallbackResponse{
			CallbackID: c.Callback().ID,
			Text:       "Entry was mark as scrolled.",
		})
	})
	b.Handle(formCallbackQuery(rateText), func(c tele.Context) error {
		parts := strings.Split(c.Callback().Data, "|")
		if len(parts) < 2 {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during restoring entry: wrong callback data"),
			})
		}
		entryID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during restoring entry: %v", err),
			})
		}
		ratingTag := parts[1]
		err = wallabagClient.AddTagsToArticle(int(entryID), []string{ratingTag})
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during restoring entry: %v", err),
			})
		}
		c.Bot().EditReplyMarkup(c.Update().Callback.Message, formInlineButtons(int(entryID), parts[2] == "1"))
		return c.Respond(&tele.CallbackResponse{
			CallbackID: c.Callback().ID,
			Text:       fmt.Sprintf("Entry was rated as %s.", ratingTag),
		})
	})
	b.Handle(tele.OnText, func(c tele.Context) error {
		c.Send("Received message, finding articles and try to save")
		for _, r := range xurls.Strict.FindAllString(c.Message().Text, -1) {
			entry, err := wallabagClient.CreateArticle(r)
			if err != nil {
				c.Send(fmt.Sprintf("Found article %s, but save failed with err: %v", r, err))
				continue
			}
			tags, err := tagger.GuessTags(entry.Title, entry.Content)
			if err != nil {
				log.Printf("error on tagging: %v\n", err)
			}
			if tags != nil {
				go wallabagClient.AddTagsToArticle(entry.ID, tags)
			}
			message := fmt.Sprintf("Found article %s and successfully saved with id: %d. Suggested tags: %v", entry.Url, entry.ID, tags)
			c.Send(formatArticleMessage(message, entry))
		}
		return nil
	})

	return b
}
