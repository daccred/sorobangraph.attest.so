package models

import "encoding/json"

type ContractEvent struct {
	ID              string          `json:"id"`
	ContractID      string          `json:"contract_id"`
	Ledger          uint32          `json:"ledger"`
	TransactionHash string          `json:"transaction_hash"`
	EventType       string          `json:"event_type"`
	Topics          []string        `json:"topics"`
	Data            json.RawMessage `json:"data"`
	InSuccessfulTx  bool            `json:"in_successful_tx"`
}