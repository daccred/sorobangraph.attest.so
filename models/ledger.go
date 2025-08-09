package models

import "time"

type LedgerInfo struct {
	Sequence        uint32    `json:"sequence"`
	Hash            string    `json:"hash"`
	PreviousHash    string    `json:"previous_hash"`
	TransactionCount int      `json:"transaction_count"`
	OperationCount  int       `json:"operation_count"`
	ClosedAt        time.Time `json:"closed_at"`
	TotalCoins      int64     `json:"total_coins"`
	FeePool         int64     `json:"fee_pool"`
	BaseFee         uint32    `json:"base_fee"`
	BaseReserve     uint32    `json:"base_reserve"`
	MaxTxSetSize    uint32    `json:"max_tx_set_size"`
	ProtocolVersion uint32    `json:"protocol_version"`
}