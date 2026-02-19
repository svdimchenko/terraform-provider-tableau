package tableau

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	ApiUrl                    string
	BaseUrl                   string
	HTTPClient                *http.Client
	AuthToken                 string
	Username                  string
	SiteID                    string
	ServerURL                 string
	Password                  string
	PersonalAccessTokenName   string
	PersonalAccessTokenSecret string
	ServerVersion             string
}

type SiteDetails struct {
	ID         *string `json:"id"`
	ContentUrl string  `json:"contentUrl"`
}

type Credentials struct {
	Name        *string     `json:"name"`
	Password    *string     `json:"password"`
	TokenName   *string     `json:"personalAccessTokenName"`
	TokenSecret *string     `json:"personalAccessTokenSecret"`
	SiteDetails SiteDetails `json:"site"`
}

type SignInRequest struct {
	Credentials Credentials `json:"credentials"`
}

type SignInResponseData struct {
	SiteDetails               SiteDetails `json:"site"`
	User                      User        `json:"user"`
	Token                     string      `json:"token"`
	EstimatedTimeToExpiration string      `json:"estimatedTimeToExpiration"`
}

type SignInResponse struct {
	SignInResponseData SignInResponseData `json:"credentials"`
}

func NewClient(server, username, password, personalAccessTokenName, personalAccessTokenSecret, site, serverVersion *string) (*Client, error) {
	c := Client{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}

	if (server != nil) && (username != nil) && (site != nil) && (serverVersion != nil) {
		baseUrl := fmt.Sprintf("%s/api/%s", *server, *serverVersion)
		url := fmt.Sprintf("%s/auth/signin", baseUrl)

		siteStruct := SiteDetails{ContentUrl: *site}
		credentials := Credentials{
			Name:        username,
			Password:    password,
			TokenName:   personalAccessTokenName,
			TokenSecret: personalAccessTokenSecret,
			SiteDetails: siteStruct,
		}
		authRequest := SignInRequest{
			Credentials: credentials,
		}
		authRequestJson, err := json.Marshal(authRequest)
		if err != nil {
			return nil, err
		}

		// authenticate
		req, err := http.NewRequest("POST", url, strings.NewReader(string(authRequestJson)))
		if err != nil {
			return nil, err
		}

		body, err := c.doRequest(req)
		if err != nil {
			return nil, err
		}

		// parse response body
		ar := SignInResponse{}
		err = json.Unmarshal(body, &ar)
		if err != nil {
			return nil, err
		}

		c.BaseUrl = baseUrl
		c.ApiUrl = fmt.Sprintf("%s/sites/%s", baseUrl, *ar.SignInResponseData.SiteDetails.ID)
		c.AuthToken = ar.SignInResponseData.Token
		c.Username = *username
		c.SiteID = *ar.SignInResponseData.SiteDetails.ID
		c.ServerURL = *server
		c.ServerVersion = *serverVersion
		if password != nil {
			c.Password = *password
		}
		if personalAccessTokenName != nil {
			c.PersonalAccessTokenName = *personalAccessTokenName
		}
		if personalAccessTokenSecret != nil {
			c.PersonalAccessTokenSecret = *personalAccessTokenSecret
		}
	}

	return &c, nil
}

// NewSiteAuthenticatedClient creates a new client authenticated to a specific site.
func (c *Client) NewSiteAuthenticatedClient(siteID string) (*Client, error) {
	newClient := Client{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}

	baseUrl := fmt.Sprintf("%s/api/%s", c.ServerURL, c.ServerVersion)
	url := fmt.Sprintf("%s/auth/signin", baseUrl)

	// Get site details first
	site, err := c.GetSite(siteID)
	if err != nil {
		return nil, err
	}

	siteStruct := SiteDetails{ContentUrl: site.ContentURL}
	credentials := Credentials{
		Name:        &c.Username,
		SiteDetails: siteStruct,
	}

	if c.Password != "" {
		credentials.Password = &c.Password
	}
	if c.PersonalAccessTokenName != "" {
		credentials.TokenName = &c.PersonalAccessTokenName
	}
	if c.PersonalAccessTokenSecret != "" {
		credentials.TokenSecret = &c.PersonalAccessTokenSecret
	}

	authRequest := SignInRequest{
		Credentials: credentials,
	}
	authRequestJson, err := json.Marshal(authRequest)
	if err != nil {
		return nil, err
	}

	// authenticate
	req, err := http.NewRequest("POST", url, strings.NewReader(string(authRequestJson)))
	if err != nil {
		return nil, err
	}

	body, err := newClient.doRequest(req)
	if err != nil {
		return nil, err
	}

	// parse response body
	ar := SignInResponse{}
	err = json.Unmarshal(body, &ar)
	if err != nil {
		return nil, err
	}

	newClient.BaseUrl = baseUrl
	newClient.ApiUrl = fmt.Sprintf("%s/sites/%s", baseUrl, *ar.SignInResponseData.SiteDetails.ID)
	newClient.AuthToken = ar.SignInResponseData.Token
	newClient.Username = c.Username
	newClient.SiteID = *ar.SignInResponseData.SiteDetails.ID
	newClient.ServerURL = c.ServerURL
	newClient.ServerVersion = c.ServerVersion
	newClient.Password = c.Password
	newClient.PersonalAccessTokenName = c.PersonalAccessTokenName
	newClient.PersonalAccessTokenSecret = c.PersonalAccessTokenSecret

	return &newClient, nil
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Tableau-Auth", c.AuthToken)

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if (res.StatusCode != http.StatusOK) && (res.StatusCode != 201) && (res.StatusCode != 204) && (res.StatusCode != 202) {
		return nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, body)
	}

	return body, err
}

// NewSiteClient creates a new client authenticated to a specific site.
func (c *Client) NewSiteClient(siteID string) (*Client, error) {
	siteClient := &Client{
		HTTPClient: c.HTTPClient,
		BaseUrl:    c.BaseUrl,
		ApiUrl:     fmt.Sprintf("%s/sites/%s", c.BaseUrl, siteID),
		AuthToken:  c.AuthToken,
		Username:   c.Username,
		SiteID:     siteID,
	}
	return siteClient, nil
}

// GetCurrentUser returns the current authenticated user.
func (c *Client) GetCurrentUser() (*User, error) {
	users, err := c.GetUsers()
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		if user.Name == c.Username {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("current user not found")
}
