package mathoverflow

// Question is the record emitted for a MathOverflow question.
type Question struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Score       int    `json:"score"`
	ViewCount   int    `json:"view_count"`
	AnswerCount int    `json:"answer_count"`
	IsAnswered  bool   `json:"is_answered"`
	Tags        string `json:"tags"`
	CreatedAt   string `json:"created_at"`
	URL         string `json:"url"`
}

// Answer is the record emitted for a MathOverflow answer.
type Answer struct {
	ID         int    `json:"id"`
	Score      int    `json:"score"`
	IsAccepted bool   `json:"is_accepted"`
	CreatedAt  string `json:"created_at"`
	URL        string `json:"url"`
}

// Tag is the record emitted for a MathOverflow tag.
type Tag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
