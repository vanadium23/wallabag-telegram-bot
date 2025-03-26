package usecase

type Rating string

const (
	Bad    Rating = "bad"
	Normal Rating = "normal"
	Good   Rating = "good"
	Great  Rating = "great"
)

func (r Rating) IsValid() bool {
	switch r {
	case Bad, Normal, Good, Great:
		return true
	}
	return false
}

func RatingFromString(s string) (Rating, bool) {
	r := Rating(s)
	return r, r.IsValid()
}
