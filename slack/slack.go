package slack

import (
	"encoding/json"
	"errors"
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
	req, err := c.newRequest("GET", "/team.profile.get")
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	profileResponse := struct {
		Profile *TeamProfile `json:"profile,omitempty"`
	}{}

	if err = json.NewDecoder(resp.Body).Decode(&profileResponse); err != nil {
		return nil, errors.New("Bad api body response")
	}

	return profileResponse.Profile, nil
}

func (c Client) GetUserProfile(id string) (*UserProfile, error) {
	req, err := c.newRequest("GET", "/users.profile.get")
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("user", id)
	req.URL.RawQuery = q.Encode()

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var profileResponse struct {
		UserProfile *UserProfile `json:"profile,omitempty"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&profileResponse); err != nil {
		return nil, errors.New("Bad api body response")
	}

	return profileResponse.UserProfile, nil
}

func (c Client) GetUsers() ([]User, error) {
	req, err := c.newRequest("GET", "/users.list")
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var usersResponse struct {
		Members []User `json:"members,omitempty"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&usersResponse); err != nil {
		return nil, errors.New("Bad api body response")
	}

	return usersResponse.Members, nil
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

func (c Client) newRequest(method, path string) (*http.Request, error) {
	req, err := http.NewRequest(method, SLACK_API+path, nil)
	if err != nil {
		return nil, err
	}

	values := url.Values{"token": {c.token}}
	req.URL.RawQuery = values.Encode()
	return req, nil
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UserProfile struct {
	Fields map[string]UserProfileField `json:"fields"`
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
