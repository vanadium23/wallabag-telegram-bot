package articles

import (
	"database/sql"
)

type ArticleRepository struct {
	db *sql.DB
}

type ArticleEntry struct {
	URL       string
	ChatID    int64
	MessageID int
	Saved     bool
}

func NewArticleRepo(db *sql.DB) (ArticleRepository, error) {
	var statement *sql.Stmt
	var err error

	ar := ArticleRepository{
		db: db,
	}

	if statement, err = db.Prepare(`CREATE TABLE IF NOT EXISTS Requests (	
		id INTEGER PRIMARY KEY AUTOINCREMENT, 
		URL TEXT, 
		ChatID INTEGER,
		MessageID INTEGER,
		saved INTEGER)`); err != nil {
		return ar, err
	}

	if _, err = statement.Exec(); err != nil {
		return ar, err
	}

	if statement, err = db.Prepare(`CREATE INDEX IF NOT EXISTS URLIndex ON Requests(URL);`); err != nil {
		return ar, err
	}

	if _, err = statement.Exec(); err != nil {
		return ar, err
	}

	return ar, nil
}

func (ar ArticleRepository) Insert(articleURL string, chatID int64, messageID int) error {
	var statement *sql.Stmt
	var err error
	statement, err = ar.db.Prepare("INSERT INTO Requests (URL, ChatID, MessageID, saved) VALUES (?, ?, ?, 0)")
	if err != nil {
		return err
	}
	_, err = statement.Exec(articleURL, chatID, messageID)
	if err != nil {
		return err
	}
	return nil
}

func (ar ArticleRepository) FetchUnsaved() ([]ArticleEntry, error) {
	var err error
	var rows *sql.Rows
	if rows, err = ar.db.Query(`
SELECT URL, ChatID, MessageID
FROM Requests
WHERE saved == 0
`); err != nil {
		return nil, err
	}

	var URL string
	var ChatID int64
	var MessageID int

	articles := make([]ArticleEntry, 0)

	for rows.Next() {
		if err := rows.Scan(&URL, &ChatID, &MessageID); err != nil {
			return nil, err
		}
		article := ArticleEntry{
			URL:       URL,
			ChatID:    ChatID,
			MessageID: MessageID,
		}
		articles = append(articles, article)
	}
	return articles, nil
}

func (ar ArticleRepository) Save(articleURL string) error {
	statement, err := ar.db.Prepare("UPDATE Requests SET saved = 1 WHERE URL = ?")
	if err != nil {
		return err
	}
	_, err = statement.Exec(articleURL)
	if err != nil {
		return err
	}
	return nil
}

func (ar ArticleRepository) CountArticleByURL(articleURL string) (int, error) {
	var count int
	row := ar.db.QueryRow("SELECT COUNT(*) FROM Requests WHERE URL = ?", articleURL)
	err := row.Scan(&count)

	if err != nil {
		return count, err
	}
	return count, nil
}
