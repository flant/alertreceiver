package webhook

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"alertreceiver/pkg/logging"
	"alertreceiver/pkg/madison"

	log "github.com/sirupsen/logrus"
)

type Handler struct {
	madisonClient *madison.Client
	logger        *logging.Logger
	dms           string
}

type AlertmanagerWebhook struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
}

type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

func NewHandler(madisonClient *madison.Client, logger *logging.Logger, dms string) *Handler {
	return &Handler{
		madisonClient: madisonClient,
		logger:        logger,
		dms:           dms,
	}
}

func (h *Handler) HandlePrometheus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body", log.Fields{"error": err.Error()})
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var webhook AlertmanagerWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		h.logger.Error("failed to unmarshal webhook", log.Fields{"error": err.Error(), "body": string(body)})
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	webhookJSON, _ := json.Marshal(webhook)
	h.logger.Info("received alertmanager webhook", log.Fields{
		"version":     webhook.Version,
		"groupKey":    webhook.GroupKey,
		"status":      webhook.Status,
		"receiver":    webhook.Receiver,
		"alertsCount": len(webhook.Alerts),
		"webhook":     string(webhookJSON),
		"timestamp":   time.Now().Format(time.RFC3339),
	})

	for _, alert := range webhook.Alerts {
		alertName := alert.Labels["alertname"]
		if alertName == "" {
			alertName = "Unknown"
		}

		severity := alert.Labels["severity"]
		if severity == "" {
			severity = "5"
		}

		severityLevel := mapSeverityToLevel(severity)

		summary := alert.Annotations["summary"]
		if summary == "" {
			summary = alertName
		}

		description := buildDescription(alert, webhook)

		grafanaURL := alert.Annotations["dashboard"]
		if grafanaURL == "" {
			grafanaURL = alert.GeneratorURL
		}

		trigger := alertName

		alertStatus := alert.Status
		if alertStatus == "resolved" {
			trigger = trigger + "_Resolved"
			if !strings.Contains(strings.ToLower(summary), "resolved") && !strings.Contains(strings.ToLower(summary), "решено") {
				summary = "✅ Resolved: " + summary
			}
		}

		alertJSON, _ := json.Marshal(alert)
		h.logger.Info("processing alert", log.Fields{
			"alertName": alertName,
			"status":    alertStatus,
			"severity":  severity,
			"trigger":   trigger,
			"summary":   summary,
			"alert":     string(alertJSON),
			"timestamp": time.Now().Format(time.RFC3339),
		})

		madisonPayload := map[string]interface{}{
			"labels": map[string]string{
				"trigger":        trigger,
				"severity_level": severityLevel,
				"alertreceiver":  "alertreceiver",
				"grafana":        grafanaURL,
			},
			"annotations": map[string]string{
				"summary":     summary,
				"description": description,
			},
		}
		madisonPayloadJSON, _ := json.Marshal(madisonPayload)

		if err := h.madisonClient.SendAlert(trigger, severityLevel, summary, description, grafanaURL); err != nil {
			h.logger.Error("failed to send alert to madison", log.Fields{
				"error":     err.Error(),
				"alertName": alertName,
				"trigger":   trigger,
				"payload":   string(madisonPayloadJSON),
				"timestamp": time.Now().Format(time.RFC3339),
			})
		} else {
			h.logger.Info("alert sent to madison", log.Fields{
				"alertName": alertName,
				"trigger":   trigger,
				"payload":   string(madisonPayloadJSON),
				"timestamp": time.Now().Format(time.RFC3339),
			})
		}
	}

	w.WriteHeader(http.StatusOK)
}

func buildDescription(alert Alert, webhook AlertmanagerWebhook) string {
	var parts []string

	if desc := alert.Annotations["description"]; desc != "" {
		desc = removeEmojis(desc)
		parts = append(parts, desc)
	}

	if dashboard := alert.Annotations["dashboard"]; dashboard != "" {
		parts = append(parts, dashboard)
	}

	if url := alert.GeneratorURL; url != "" {
		parts = append(parts, url)
	}

	if len(parts) == 0 {
		return "No description provided"
	}

	return strings.Join(parts, "\n\n")
}

func removeEmojis(s string) string {
	var result strings.Builder
	for _, r := range s {
		if !isEmoji(r) {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}

func isEmoji(r rune) bool {
	return (r >= 0x1F600 && r <= 0x1F64F) ||
		(r >= 0x1F300 && r <= 0x1F5FF) ||
		(r >= 0x1F680 && r <= 0x1F6FF) ||
		(r >= 0x1F1E0 && r <= 0x1F1FF) ||
		(r >= 0x2600 && r <= 0x26FF) ||
		(r >= 0x2700 && r <= 0x27BF) ||
		(r >= 0xFE00 && r <= 0xFE0F) ||
		(r >= 0x1F900 && r <= 0x1F9FF) ||
		(r >= 0x1FA00 && r <= 0x1FA6F) ||
		r == 0x200D ||
		r == 0xFE0F
}

func mapSeverityToLevel(severity string) string {
	severityMap := map[string]string{
		"critical": "1",
		"warning":  "3",
		"info":     "5",
	}

	level := severityMap[strings.ToLower(severity)]
	if level == "" {
		return "5"
	}
	return level
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handler) SendDMS() {
	h.madisonClient.SendDeadMansSwitch(h.dms)
}
