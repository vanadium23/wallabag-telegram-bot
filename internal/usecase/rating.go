package usecase

type Rating string

const (
	Bad    Rating = "rating:bad"
	Normal Rating = "rating:normal"
	Good   Rating = "rating:good"
	Great  Rating = "rating:great"
)

func (r Rating) IsValid() bool {
	switch r {
	case Bad, Normal, Good, Great:
		return true
	// previous versions
	case "bad", "normal", "good", "great":
		return true
	}
	return false
}

func RatingFromString(s string) (Rating, bool) {
	r := Rating(s)
	return r, r.IsValid()
}
