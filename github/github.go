package github

type Repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

type PullRequest struct {
	HTMLURL string `json:"html_url"`
	Number  int    `json:"number"`
	Title   string `json:"title"`
	User    *User  `json:"user"`
}

type User struct {
	Login string `json:"login"`
	ID    int    `json:"id"`
}
