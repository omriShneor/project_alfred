package database

import (
	"encoding/json"
	"fmt"
)

// AnalysisTrace captures model routing/analyzer decisions for quality measurement.
type AnalysisTrace struct {
	UserID           int64
	ChannelID        int64
	SourceType       string
	TriggerMessageID *int64
	Intent           string
	RouterConfidence float64
	Action           string
	Confidence       float64
	Reasoning        string
	Status           string
	Details          map[string]any
}

func (d *DB) CreateAnalysisTrace(trace AnalysisTrace) error {
	detailsJSON := "{}"
	if len(trace.Details) > 0 {
		if b, err := json.Marshal(trace.Details); err == nil {
			detailsJSON = string(b)
		}
	}

	_, err := d.Exec(`
		INSERT INTO analysis_traces (
			user_id, channel_id, source_type, trigger_message_id, intent,
			router_confidence, action, confidence, reasoning, status, details_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		trace.UserID,
		trace.ChannelID,
		trace.SourceType,
		trace.TriggerMessageID,
		trace.Intent,
		trace.RouterConfidence,
		trace.Action,
		trace.Confidence,
		trace.Reasoning,
		trace.Status,
		detailsJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create analysis trace: %w", err)
	}
	return nil
}

