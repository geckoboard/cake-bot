package github

type Repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

type Review struct {
	ID   int   `json:"id"`
	User *User `json:"user"`

	// State can be either "approved" or "change_requested".
	State string `json:"state"`

	Links Links `json:"_links"`
}

func (r *Review) IsApproved() bool {
	return r.State == "approved"
}

func (r *Review) HTMLURL() string {
	return r.Links.GetURL("html")
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

type Links map[string]struct {
	URL string `json:"href"`
}

func (ls Links) GetURL(key string) string {
	link, ok := ls[key]
	if !ok {
		return ""
	}
	return link.URL
}
