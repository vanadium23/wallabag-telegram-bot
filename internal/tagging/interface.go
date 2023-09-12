package tagging

type Tagger interface {
	GuessTags(title, content string) ([]string, error)
}
