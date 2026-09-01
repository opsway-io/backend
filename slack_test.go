package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

type SlackMessage struct {
	Blocks []map[string]interface{} `json:"blocks"`
}

func TestSlackBlocks(t *testing.T) {
	jsonStr := `{
		"blocks": [
			{"type": "header", "text": {"type": "plain_text", "text": "🚨 Incident Alert", "emoji": true}},
			{"type": "actions", "elements": [
				{"type": "button", "action_id": "acknowledge_incident", "value": "ack_1"},
				{"type": "button", "action_id": "resolve_incident", "value": "res_1"}
			]}
		]
	}`
	var msg SlackMessage
	json.Unmarshal([]byte(jsonStr), &msg)

	blocks := msg.Blocks
	isAck := true
	statusText := "👀 Incident Acknowledged"

	if len(blocks) > 0 {
		if blocks[0]["type"] == "header" {
			blocks[0]["text"] = map[string]interface{}{
				"type":  "plain_text",
				"text":  statusText,
				"emoji": true,
			}
		}
	}

	if isAck {
		for i, block := range blocks {
			if block["type"] == "actions" {
				if elements, ok := block["elements"].([]interface{}); ok {
					newElements := []interface{}{}
					for _, el := range elements {
						if elMap, ok := el.(map[string]interface{}); ok {
							if actionID, _ := elMap["action_id"].(string); actionID != "acknowledge_incident" {
								newElements = append(newElements, el)
							}
						}
					}
					blocks[i]["elements"] = newElements
				}
			}
		}
	} else {
		newBlocks := []map[string]interface{}{}
		for _, block := range blocks {
			if block["type"] != "actions" {
				newBlocks = append(newBlocks, block)
			}
		}
		blocks = newBlocks
	}

	out, _ := json.Marshal(blocks)
	fmt.Println(string(out))
}
