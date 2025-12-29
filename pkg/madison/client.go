package madison

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"alertreceiver/pkg/config"
)

type Client struct {
	apiKey     string
	madisonURL string
	httpClient *http.Client
}

type Alert struct {
	Labels      Labels      `json:"labels"`
	Annotations Annotations `json:"annotations"`
}

type Labels map[string]string

type Annotations struct {
	Summary     string `json:"summary"`
	Description string `json:"description"`
}

type Dms struct {
	DmsLabels DmsLabels `json:"labels"`
}

type DmsLabels struct {
	Trigger string `json:"trigger"`
	Dms     string `json:"dms"`
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		apiKey:     cfg.MadisonAPIKey,
		madisonURL: cfg.MadisonURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SendAlert(trigger, severity, summary, description, grafana string, alertLabels map[string]string) error {
	labels := make(Labels)

	for k, v := range alertLabels {
		labels[k] = v
	}

	severityLevel := severity
	if severityLevel != "" {
		level, err := strconv.Atoi(severityLevel)
		if err == nil {
			level++
			if level > 5 {
				level = 5
			}
			severityLevel = strconv.Itoa(level)
		} else {
			severityLevel = "5"
		}
	} else {
		severityLevel = "5"
	}

	labels["trigger"] = trigger
	labels["severity_level"] = severityLevel
	labels["alertreceiver"] = "alertreceiver"

	payload := Alert{
		Labels: labels,
		Annotations: Annotations{
			Summary:     summary,
			Description: description,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %v", err)
	}

	requestURL := fmt.Sprintf(c.madisonURL, c.apiKey)

	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return fmt.Errorf("madison API returned 403 Forbidden - check key and permissions")
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("madison API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) SendDeadMansSwitch(dms string) {
	payload := Dms{
		DmsLabels: DmsLabels{
			Trigger: "DeadMansSwitch",
			Dms:     dms,
		},
	}

	jsonData, _ := json.Marshal(payload)
	requestURL := fmt.Sprintf(c.madisonURL, c.apiKey)

	req, _ := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := c.httpClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
}
