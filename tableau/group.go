package tableau

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type GroupImport struct {
	Source           *string `json:"source,omitempty"`
	DomainName       *string `json:"domainName,omitempty"`
	MinimumSiteRole  *string `json:"siteRole,omitempty"`
	GrantLicenseMode *string `json:"grantLicenseMode,omitempty"`
}

type Job struct {
	ID         string `json:"id"`
	Mode       string `json:"mode"`
	Type       string `json:"type"`
	Progress   string `json:"progress"`
	CreatedAt  string `json:"createdAt"`
	FinishCode string `json:"finishCode"`
}

type JobResponse struct {
	Job Job `json:"job"`
}

type Group struct {
	ID              string       `json:"id,omitempty"`
	Name            string       `json:"name"`
	MinimumSiteRole string       `json:"minimumSiteRole,omitempty"`
	Import          *GroupImport `json:"import,omitempty"`
}

type GroupRequest struct {
	Group Group `json:"group"`
}

type GroupResponse struct {
	Group Group `json:"group"`
}

type GroupsResponse struct {
	Groups []Group `json:"group"`
}

type GroupListResponse struct {
	GroupsResponse GroupsResponse    `json:"groups"`
	Pagination     PaginationDetails `json:"pagination"`
}

func (c *Client) GetGroups() ([]Group, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/groups", c.ApiUrl), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	groupListResponse := GroupListResponse{}
	err = json.Unmarshal(body, &groupListResponse)
	if err != nil {
		return nil, err
	}

	// TODO: Generalise pagination handling and use elsewhere
	pageNumber, totalPageCount, totalAvailable, err := GetPaginationNumbers(groupListResponse.Pagination)
	if err != nil {
		return nil, err
	}

	allGroups := make([]Group, 0, totalAvailable)
	allGroups = append(allGroups, groupListResponse.GroupsResponse.Groups...)

	for page := pageNumber + 1; page <= totalPageCount; page++ {
		fmt.Printf("Searching page %d", page)
		req, err = http.NewRequest("GET", fmt.Sprintf("%s/groups?pageNumber=%d", c.ApiUrl, page), nil)
		if err != nil {
			return nil, err
		}
		body, err = c.doRequest(req)
		if err != nil {
			return nil, err
		}
		groupListResponse = GroupListResponse{}
		err = json.Unmarshal(body, &groupListResponse)
		if err != nil {
			return nil, err
		}
		allGroups = append(allGroups, groupListResponse.GroupsResponse.Groups...)
	}

	return allGroups, nil
}

func (c *Client) GetGroup(groupID string) (*Group, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/groups", c.ApiUrl), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	groupListResponse := GroupListResponse{}
	err = json.Unmarshal(body, &groupListResponse)
	if err != nil {
		return nil, err
	}

	// TODO: Generalise pagination handling and use elsewhere
	pageNumber, totalPageCount, _, err := GetPaginationNumbers(groupListResponse.Pagination)
	if err != nil {
		return nil, err
	}
	for i, group := range groupListResponse.GroupsResponse.Groups {
		if group.ID == groupID {
			return &groupListResponse.GroupsResponse.Groups[i], nil
		}
	}

	for page := pageNumber + 1; page <= totalPageCount; page++ {
		fmt.Printf("Searching page %d", page)
		req, err = http.NewRequest("GET", fmt.Sprintf("%s/groups?pageNumber=%d", c.ApiUrl, page), nil)
		if err != nil {
			return nil, err
		}
		body, err = c.doRequest(req)
		if err != nil {
			return nil, err
		}
		groupListResponse = GroupListResponse{}
		err = json.Unmarshal(body, &groupListResponse)
		if err != nil {
			return nil, err
		}
		// check if we found the group in this page
		for i, group := range groupListResponse.GroupsResponse.Groups {
			if group.ID == groupID {
				return &groupListResponse.GroupsResponse.Groups[i], nil
			}
		}
	}

	return nil, fmt.Errorf("did not find group ID %s", groupID)
}

func (c *Client) CreateGroup(name, minimumSiteRole string) (*Group, error) {

	newGroup := Group{
		Name:            name,
		MinimumSiteRole: minimumSiteRole,
	}
	groupRequest := GroupRequest{
		Group: newGroup,
	}

	newGroupJson, err := json.Marshal(groupRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/groups", c.ApiUrl), strings.NewReader(string(newGroupJson)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	groupResponse := GroupResponse{}
	err = json.Unmarshal(body, &groupResponse)
	if err != nil {
		return nil, err
	}

	return &groupResponse.Group, nil
}

func (c *Client) UpdateGroup(groupID, name, minimumSiteRole string) (*Group, error) {

	group := Group{
		Name:            name,
		MinimumSiteRole: minimumSiteRole,
	}
	groupRequest := GroupRequest{
		Group: group,
	}

	updateGroupJson, err := json.Marshal(groupRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/groups/%s", c.ApiUrl, groupID), strings.NewReader(string(updateGroupJson)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	groupResponse := GroupResponse{}
	err = json.Unmarshal(body, &groupResponse)
	if err != nil {
		return nil, err
	}

	return &groupResponse.Group, nil
}

func (c *Client) DeleteGroup(groupID string) error {

	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/groups/%s", c.ApiUrl, groupID), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) ImportGroup(name, domainName, minimumSiteRole, grantLicenseMode string, asyncMode bool) (*Group, error) {
	source := "ActiveDirectory"

	if domainName == "" {
		parts := strings.Split(name, "\\")
		if len(parts) == 2 {
			domainName = parts[0]
		}
	}

	newGroup := Group{
		Name: name,
		Import: &GroupImport{
			Source:     &source,
			DomainName: &domainName,
		},
	}

	if minimumSiteRole != "" {
		newGroup.Import.MinimumSiteRole = &minimumSiteRole
	}
	if grantLicenseMode != "" {
		newGroup.Import.GrantLicenseMode = &grantLicenseMode
	}

	groupRequest := GroupRequest{
		Group: newGroup,
	}

	newGroupJson, err := json.Marshal(groupRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/groups?asJob=true", c.ApiUrl), strings.NewReader(string(newGroupJson)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	jobResponse := JobResponse{}
	err = json.Unmarshal(body, &jobResponse)
	if err != nil {
		return nil, err
	}
	err = c.WaitForJob(jobResponse.Job.ID)
	if err != nil {
		return nil, err
	}
	groups, err := c.GetGroups()
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		if group.Name == name {
			return &group, nil
		}
	}
	return nil, fmt.Errorf("group %s not found after job completion", name)
}

func (c *Client) WaitForJob(jobID string) error {
	for i := 0; i < 60; i++ {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/jobs/%s", c.ApiUrl, jobID), nil)
		if err != nil {
			return err
		}
		body, err := c.doRequest(req)
		if err != nil {
			return err
		}
		jobResponse := JobResponse{}
		err = json.Unmarshal(body, &jobResponse)
		if err != nil {
			return err
		}
		if jobResponse.Job.Progress == "100" {
			if jobResponse.Job.FinishCode == "0" {
				return nil
			}
			return fmt.Errorf("job failed with finish code %s", jobResponse.Job.FinishCode)
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("job timeout after 60 attempts")
}
