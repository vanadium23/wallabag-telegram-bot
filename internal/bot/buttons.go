package bot

import (
	"strconv"

	tele "gopkg.in/telebot.v3"
)

func formInlineButtons(entryID int, archive bool) *tele.ReplyMarkup {
	// buttons
	entry := strconv.Itoa(entryID)
	selector := &tele.ReplyMarkup{}
	stateBtn := tele.Btn{}
	if archive {
		stateBtn = selector.Data("✅", archiveText, entry)
	} else {
		stateBtn = selector.Data("📥", unarchiveText, entry)
	}
	// scrolled
	archiveText := "0"
	if archive {
		archiveText = "1"
	}
	scrolledButton := selector.Data("📜", scrolledText, entry, archiveText)
	// rating
	var EmojiToRating = map[string]string{
		"😡": "bad",
		"😕": "normal",
		"😊": "good",
		"😎": "great",
	}
	ratingRow := selector.Row()
	for emoji, rating := range EmojiToRating {
		btn := selector.Data(emoji, rateText, entry, rating, archiveText)
		ratingRow = append(ratingRow, btn)
	}
	selector.Inline(
		selector.Row(stateBtn, scrolledButton),
		ratingRow,
	)
	return selector
}
