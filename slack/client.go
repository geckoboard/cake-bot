package slack

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
)

var SLACK_API string = "https://slack.com/api"
var HTTPClient = &http.Client{}

func New(token string) *Client {
	return &Client{token}
}

type Client struct {
	token string
}

func (c Client) GetTeamProfile() (*TeamProfile, error) {
	req, err := c.newRequest("GET", "/team.profile.get", nil)
	if err != nil {
		return nil, err
	}

	values := url.Values{"token": {c.token}}
	req.URL.RawQuery = values.Encode()

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response TeamProfileResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if !response.OK {
		return nil, errors.New(response.Error)
	}

	return &response.TeamProfile, nil
}

func (c Client) GetUserProfile(id string) (*UserProfile, error) {
	req, err := c.newRequest("GET", "/users.profile.get", nil)
	if err != nil {
		return nil, err
	}

	values := url.Values{
		"token": {c.token},
		"user":  {id},
	}
	req.URL.RawQuery = values.Encode()

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response UserProfileResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if !response.OK {
		return nil, errors.New(response.Error)
	}

	return &response.UserProfile, nil
}

func (c Client) GetUsers() ([]User, error) {
	req, err := c.newRequest("GET", "/users.list", nil)
	if err != nil {
		return nil, err
	}

	values := url.Values{
		"token": {c.token},
	}
	req.URL.RawQuery = values.Encode()

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response UsersResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if !response.OK {
		return nil, errors.New(response.Error)
	}

	return response.Users, nil
}

func (c Client) doRequest(req *http.Request) (*http.Response, error) {
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, errors.New("Bad api status response")
	}

	return resp, nil
}

func (c Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, SLACK_API+path, body)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (p *UserProfile) UnmarshalJSON(data []byte) error {
	var userProfile1 struct {
		Fields map[string]UserProfileField `json:"fields"`
	}

	var userProfile2 struct {
		Fields []interface{} `json:"fields"`
	}

	if err := json.Unmarshal(data, &userProfile2); err != nil {
		if err = json.Unmarshal(data, &userProfile1); err != nil {
			return err
		} else {
			p.Fields = userProfile1.Fields
		}
	}

	return nil
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UserProfile struct {
	Fields map[string]UserProfileField `json:"fields"`
}

type UserProfileField struct {
	Value string `json:"value"`
}

type TeamProfile struct {
	Fields []TeamProfileField `json:"fields"`
}

type TeamProfileField struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type SlackResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

type TeamProfileResponse struct {
	TeamProfile TeamProfile `json:"profile"`
	SlackResponse
}

type UserProfileResponse struct {
	UserProfile UserProfile `json:"profile"`
	SlackResponse
}

type UsersResponse struct {
	Users []User `json:"members"`
	SlackResponse
}
