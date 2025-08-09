package models

import "encoding/json"

type Operation struct {
	ID            string          `json:"id"`
	TransactionID string          `json:"transaction_id"`
	Index         uint32          `json:"index"`
	Type          string          `json:"type"`
	SourceAccount string          `json:"source_account,omitempty"`
	Details       json.RawMessage `json:"details"`
}