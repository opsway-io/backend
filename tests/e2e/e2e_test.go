//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"
)

const (
	apiURL     = "http://localhost:8001/v1"
	mockURL    = "http://host.docker.internal:8080"
	mailHogURL = "http://localhost:8025/api/v2/messages"
)

func TestE2E_MockServer(t *testing.T) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// 1. Authentication
	t.Log("Authenticating as John Doe...")
	loginBody := map[string]string{
		"email":    "john@opsway.io",
		"password": "pass",
	}
	loginJSON, _ := json.Marshal(loginBody)
	resp, err := client.Post(apiURL+"/auth/login", "application/json", bytes.NewBuffer(loginJSON))
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Login failed with status %d", resp.StatusCode)
	}

	var loginResp struct {
		User struct {
			Teams []struct {
				ID uint `json:"id"`
			} `json:"teams"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("Failed to decode login response: %v", err)
	}

	if len(loginResp.User.Teams) == 0 {
		t.Fatalf("User has no teams")
	}
	teamID := loginResp.User.Teams[0].ID
	t.Logf("Authenticated successfully. Team ID: %d", teamID)

	// 2. Create Monitor 1: Status 500 (Incident Test)
	t.Log("Creating monitor for Status 500...")
	monitor500Body := map[string]interface{}{
		"name": "E2E Status 500 Monitor",
		"settings": map[string]interface{}{
			"method":           "GET",
			"url":              mockURL + "/status/500",
			"frequencySeconds": 30,
			"headers":          []interface{}{},
			"body": map[string]interface{}{
				"type": "NONE",
			},
			"tls": map[string]interface{}{
				"enabled": false,
			},
			"locations": []string{"global"},
		},
		"assertions": []map[string]interface{}{
			{
				"source":   "STATUS_CODE",
				"property": "",
				"operator": "EQUAL",
				"target":   "200",
			},
		},
	}
	monitor500JSON, _ := json.Marshal(monitor500Body)
	resp500, err := client.Post(fmt.Sprintf("%s/teams/%d/monitors", apiURL, teamID), "application/json", bytes.NewBuffer(monitor500JSON))
	if err != nil {
		t.Fatalf("Failed to create monitor 500: %v", err)
	}
	defer resp500.Body.Close()
	if resp500.StatusCode > 299 {
		body, _ := io.ReadAll(resp500.Body)
		t.Fatalf("Create monitor 500 failed with status %d: %s", resp500.StatusCode, string(body))
	}

	// 3. Create Monitor 2: Delay 5s (Anomaly Test)
	t.Log("Creating monitor for Delay 4s...")
	monitorDelayBody := map[string]interface{}{
		"name": "E2E Anomaly Monitor",
		"settings": map[string]interface{}{
			"method":           "GET",
			"url":              mockURL + "/delay/4",
			"frequencySeconds": 30,
			"headers":          []interface{}{},
			"body": map[string]interface{}{
				"type": "NONE",
			},
			"tls": map[string]interface{}{
				"enabled": false,
			},
			"locations": []string{"global"},
		},
		"assertions": []map[string]interface{}{
			{
				"source":   "STATUS_CODE",
				"property": "",
				"operator": "EQUAL",
				"target":   "200",
			},
		},
	}
	monitorDelayJSON, _ := json.Marshal(monitorDelayBody)
	respDelay, err := client.Post(fmt.Sprintf("%s/teams/%d/monitors", apiURL, teamID), "application/json", bytes.NewBuffer(monitorDelayJSON))
	if err != nil {
		t.Fatalf("Failed to create delay monitor: %v", err)
	}
	defer respDelay.Body.Close()
	if respDelay.StatusCode > 299 {
		body, _ := io.ReadAll(respDelay.Body)
		t.Fatalf("Create delay monitor failed with status %d: %s", respDelay.StatusCode, string(body))
	}

	// 4. Wait for Prober
	t.Log("Waiting 35 seconds for prober to execute checks...")
	time.Sleep(35 * time.Second)

	// 5. Verify Incidents
	t.Log("Fetching incidents...")
	respIncidents, err := client.Get(fmt.Sprintf("%s/teams/%d/incidents", apiURL, teamID))
	if err != nil {
		t.Fatalf("Failed to get incidents: %v", err)
	}
	defer respIncidents.Body.Close()
	if respIncidents.StatusCode != http.StatusOK {
		t.Fatalf("Get incidents failed with status %d", respIncidents.StatusCode)
	}

	var incidentsResp struct {
		Incidents []struct {
			Title string `json:"title"`
		} `json:"incidents"`
	}
	if err := json.NewDecoder(respIncidents.Body).Decode(&incidentsResp); err != nil {
		t.Fatalf("Failed to decode incidents response: %v", err)
	}

	foundStatusCodeIncident := false
	foundAnomalyIncident := false
	t.Log("Incidents returned by API:")
	for _, inc := range incidentsResp.Incidents {
		t.Logf(" - %s", inc.Title)
		if inc.Title == "STATUS_CODE" {
			foundStatusCodeIncident = true
		}
		if inc.Title == "Anomaly Detected" {
			foundAnomalyIncident = true
		}
	}

	if !foundStatusCodeIncident {
		t.Errorf("Expected STATUS_CODE incident to be created, but it was not found.")
	}
	if !foundAnomalyIncident {
		t.Errorf("Expected Anomaly Detected incident to be created, but it was not found.")
	}

	// 6. Verify Alerts in MailHog
	t.Log("Verifying alerts in MailHog...")
	respMail, err := http.Get(mailHogURL)
	if err != nil {
		t.Fatalf("Failed to get mailhog messages: %v", err)
	}
	defer respMail.Body.Close()

	var mailResp struct {
		Items []struct {
			Content struct {
				Headers map[string][]string `json:"Headers"`
				Body    string              `json:"Body"`
			} `json:"Content"`
		} `json:"items"`
	}
	if err := json.NewDecoder(respMail.Body).Decode(&mailResp); err != nil {
		t.Fatalf("Failed to decode mailhog response: %v", err)
	}

	if len(mailResp.Items) == 0 {
		t.Errorf("Expected emails to be sent, but MailHog mailbox is empty.")
	} else {
		t.Logf("Found %d emails in MailHog.", len(mailResp.Items))
	}
}
