package slack

import (
	"net/http"

	"github.com/slack-go/slack"
)

var SLACK_API string = "https://slack.com/api"
var HTTPClient = &http.Client{}

func New(token string) *Client {
	return &Client{
		api: slack.New(token),
	}
}

type Client struct {
	api *slack.Client
}

func (c *Client) GetTeamProfile() (*slack.TeamProfile, error) {
	return c.api.GetTeamProfile()
}

func (c *Client) GetUserProfile(id string) (*slack.UserProfile, error) {
	return c.api.GetUserProfile(&slack.GetUserProfileParameters{UserID: id})
}

func (c *Client) GetUsers() ([]slack.User, error) {
	return c.api.GetUsers()
}
