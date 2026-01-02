package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type IntervalsClient struct {
	BaseURL    string
	APIKey     string
	AthleteID  string
	HTTPClient *http.Client
}

func NewIntervalsClient(apiKey, athleteID string) *IntervalsClient {
	return &IntervalsClient{
		BaseURL:   "https://intervals.icu",
		APIKey:    apiKey,
		AthleteID: athleteID,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *IntervalsClient) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.BaseURL, path)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.SetBasicAuth("API_KEY", c.APIKey)
	return c.HTTPClient.Do(req)
}

func (c *IntervalsClient) ListActivities(oldest, newest time.Time) ([]Activity, error) {
	path := fmt.Sprintf("/api/v1/athlete/%s/activities?oldest=%s&newest=%s",
		c.AthleteID, oldest.Format("2006-01-02"), newest.Format("2006-01-02"))

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var activities []Activity
	if err := json.NewDecoder(resp.Body).Decode(&activities); err != nil {
		return nil, err
	}

	return activities, nil
}

func (c *IntervalsClient) GetActivityDetail(id string) (*ActivityDetail, error) {
	path := fmt.Sprintf("/api/v1/activity/%s", id)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var detail ActivityDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, err
	}

	return &detail, nil
}

func (c *IntervalsClient) DeleteActivity(id string) error {
	path := fmt.Sprintf("/api/v1/activity/%s", id)
	resp, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code during deletion: %d", resp.StatusCode)
	}

	return nil
}
func (c *IntervalsClient) UpdateActivity(id string, updates map[string]interface{}) error {
	path := fmt.Sprintf("/api/v1/activity/%s", id)
	body, err := json.Marshal(updates)
	if err != nil {
		return err
	}

	resp, err := c.doRequest("PUT", path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code during update: %d", resp.StatusCode)
	}

	return nil
}
