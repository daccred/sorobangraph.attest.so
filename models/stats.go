package models

import "time"

type Stats struct {
	TransactionCount int64     `json:"transaction_count"`
	EventCount       int64     `json:"event_count"`
	OperationCount   int64     `json:"operation_count"`
	CurrentLedger    uint32    `json:"current_ledger"`
	LedgersProcessed int64     `json:"ledgers_processed"`
	StartTime        time.Time `json:"start_time"`
	LastUpdateTime   time.Time `json:"last_update_time"`
	ProcessingRate   float64   `json:"processing_rate"` // ledgers per second
	ConnectedClients int       `json:"connected_clients"`
}
