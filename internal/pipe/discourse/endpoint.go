package discourse

type postsRequest struct {
	Title    string `json:"title"`
	Raw      string `json:"raw"`
	Category int    `json:"category"`
}
