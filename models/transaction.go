package models

import (
	"time"
)

type Transaction struct {
	ID             string      `json:"id"`
	Hash           string      `json:"hash"`
	Ledger         uint32      `json:"ledger"`
	Index          uint32      `json:"index"`
	SourceAccount  string      `json:"source_account"`
	FeePaid        int64       `json:"fee_paid"`
	OperationCount int32       `json:"operation_count"`
	CreatedAt      time.Time   `json:"created_at"`
	MemoType       string      `json:"memo_type,omitempty"`
	MemoValue      string      `json:"memo_value,omitempty"`
	Successful     bool        `json:"successful"`
	Operations     []Operation `json:"operations,omitempty"`
}