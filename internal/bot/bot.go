package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/vanadium23/wallabag-telegram-bot/internal/summarization"
	"github.com/vanadium23/wallabag-telegram-bot/internal/usecase"
	tele "gopkg.in/telebot.v3"
	"mvdan.cc/xurls"
)

const (
	archiveText   = "archive"
	unarchiveText = "unarchive"
	scrolledText  = "scrolled"
	rateText      = "rate"
	unrateText    = "unrate"
	summarizeText = "summarize"
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

func StartTelegramBot(
	telegramBotToken string,
	pollInterval time.Duration,
	filterUsers []string,
	// for handlers
	wallabotUseCase usecase.ArticleUseCase,
	summarizier summarization.Summarizer,
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
		articles, err := wallabotUseCase.FindRandom(5)
		if err != nil {
			log.Printf("Wallabag failed with error: %v", err)
			return c.Send("Wallabag failed with error: %v", err)
		}
		for _, article := range articles {
			msg := formatArticleMessage(article)
			btns := formArticleButtons(article)
			c.Send(msg, btns)
		}
		return nil
	})
	b.Handle("/recent", func(c tele.Context) error {
		articles, err := wallabotUseCase.FindRecent(5)
		if err != nil {
			log.Printf("Wallabag failed with error: %v", err)
			return c.Send("Wallabag failed with error: %v", err)
		}
		for _, article := range articles {
			msg := formatArticleMessage(article)
			btns := formArticleButtons(article)
			c.Send(msg, btns)
		}
		return nil
	})
	b.Handle("/short", func(c tele.Context) error {
		articles, err := wallabotUseCase.FindShort(5)
		if err != nil {
			log.Printf("Wallabag failed with error: %v", err)
			return c.Send("Wallabag failed with error: %v", err)
		}
		for _, article := range articles {
			msg := formatArticleMessage(article)
			btns := formArticleButtons(article)
			c.Send(msg, btns)
		}
		return nil
	})
	b.Handle("/stats", func(c tele.Context) error {
		stats, err := wallabotUseCase.GetStats()
		if err != nil {
			log.Printf("Wallabag failed with error: %v", err)
			return c.Send(fmt.Sprintf("Failed to get statistics: %v", err))
		}

		message := fmt.Sprintf(`
📊 Wallabag Statistics

	📚 Total unread articles: %d
	✅ Articles archived today: %d
	📅 Articles archived (last 7 days): %d
	➕ Articles added (last 7 days): %d`,
			stats.TotalUnread,
			stats.ArchivedToday,
			stats.ArchivedLast7Days,
			stats.AddedLast7Days)

		return c.Send(message)
	})
	b.Handle(formCallbackQuery(archiveText), func(c tele.Context) error {
		entryID, err := strconv.ParseInt(c.Callback().Data, 10, 64)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during archiving entry: %v", err),
			})
		}
		article, err := wallabotUseCase.MarkRead(int(entryID))
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during archiving entry: %v", err),
			})
		}
		c.Bot().EditReplyMarkup(c.Update().Callback.Message, formArticleButtons(article))
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
		article, err := wallabotUseCase.MarkUnread(int(entryID))
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during archiving entry: %v", err),
			})
		}
		c.Bot().EditReplyMarkup(c.Update().Callback.Message, formArticleButtons(article))
		return c.Respond(&tele.CallbackResponse{
			CallbackID: c.Callback().ID,
			Text:       "Entry was successfully saved back.",
		})
	})
	b.Handle(formCallbackQuery(scrolledText), func(c tele.Context) error {
		entryID, err := strconv.ParseInt(c.Callback().Data, 10, 64)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during mark as scrolled entry: %v", err),
			})
		}
		article, err := wallabotUseCase.MarkScrolled(int(entryID))
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during mark as scrolled entry: %v", err),
			})
		}
		c.Bot().EditReplyMarkup(c.Update().Callback.Message, formArticleButtons(article))
		return c.Respond(&tele.CallbackResponse{
			CallbackID: c.Callback().ID,
			Text:       "Entry was mark as scrolled and archived.",
		})
	})
	b.Handle(formCallbackQuery(rateText), func(c tele.Context) error {
		parts := strings.Split(c.Callback().Data, "|")
		if len(parts) < 2 {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during rate entry: wrong callback data"),
			})
		}
		entryID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during rate entry: %v", err),
			})
		}
		ratingTag := parts[1]
		article, err := wallabotUseCase.AddRating(int(entryID), ratingTag)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during rate entry: %v", err),
			})
		}
		c.Bot().EditReplyMarkup(c.Update().Callback.Message, formArticleButtons(article))
		return c.Respond(&tele.CallbackResponse{
			CallbackID: c.Callback().ID,
			Text:       fmt.Sprintf("Entry mark as read and was rated as %s.", ratingTag),
		})
	})
	b.Handle(formCallbackQuery(summarizeText), func(c tele.Context) error {
		entryID, err := strconv.ParseInt(c.Callback().Data, 10, 64)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during summarize entry: %v", err),
			})
		}
		article, err := wallabotUseCase.FindByID(int(entryID))
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during summarize entry: %v", err),
			})
		}
		summary, err := summarizier.Summarize(article.Title, article.Content)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				CallbackID: c.Callback().ID,
				Text:       fmt.Sprintf("Error during summarize entry: %v", err),
			})
		}
		c.Bot().Send(c.Sender(), fmt.Sprintf("Summary %d: %s", entryID, summary))
		return nil
	})

	b.Handle(tele.OnText, func(c tele.Context) error {
		c.Send("Received message, finding articles and try to save")
		for _, r := range xurls.Strict.FindAllString(c.Message().Text, -1) {
			article, err := wallabotUseCase.SaveForLater(r)
			if err != nil {
				c.Send(fmt.Sprintf("Found article %s, but save failed with err: %v", r, err))
				continue
			}
			c.Send(formatArticleMessage(article), formArticleButtons(article))
		}
		return nil
	})

	return b
}

// formCallbackQuery generates same string as InlineButton.CallbackUnique from telebot
func formCallbackQuery(text string) string {
	return "\f" + text
}

const entryMessageTemplates = `
Article №%d

### %s

%s

tags: %s

📅 %s ⏳ %d min
`

func formatArticleMessage(article usecase.WallabotArticle) string {
	return fmt.Sprintf(entryMessageTemplates,
		article.ID,
		article.Title,
		article.Url,
		article.PublicTags(),
		article.CreatedAt.Format("2006-01-02"),
		article.ReadingTime,
	)
}

func formArticleButtons(article usecase.WallabotArticle) *tele.ReplyMarkup {
	entry := strconv.Itoa(article.ID)

	selector := &tele.ReplyMarkup{}
	stateRow := selector.Row()
	stateBtn := tele.Btn{}
	if !article.IsRead {
		stateBtn = selector.Data("✅", archiveText, entry)
		summaryBtn := selector.Data("📝", summarizeText, entry)
		stateRow = append(stateRow, summaryBtn)
	} else {
		stateBtn = selector.Data("📥", unarchiveText, entry)
	}
	// scrolled
	scrolledButton := selector.Data("📜", scrolledText, entry)
	stateRow = append(stateRow, stateBtn, scrolledButton)
	// ratings
	ratingRow := selector.Row()
	if !article.HasRating {
		// rating
		var emojis = []string{"👎", "😕", "👍", "🌟"}
		var tags = []string{"bad", "normal", "good", "great"}
		for i, emoji := range emojis {
			btn := selector.Data(emoji, rateText, entry, tags[i])
			ratingRow = append(ratingRow, btn)
		}
	} else {
		unrateBtn := selector.Data("⚖️", unrateText, entry)
		stateRow = append(stateRow, unrateBtn)
	}

	selector.Inline(
		stateRow,
		ratingRow,
	)
	return selector
}
