package litefs

import (
	"encoding/json"
	"time"
)

////
// copied from litefs repo.
// modifications are commented.

const (
	EventTypeInit          = "init"
	EventTypeTx            = "tx"
	EventTypePrimaryChange = "primaryChange"
)

// Event represents a generic event.
type Event struct {
	Type string `json:"type"`
	DB   string `json:"db,omitempty"`
	Data any    `json:"data,omitempty"`
}

func (e *Event) UnmarshalJSON(data []byte) error {
	var v eventJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	e.Type = v.Type
	e.DB = v.DB

	switch v.Type {
	case EventTypeInit:
		e.Data = &InitEventData{}
	case EventTypeTx:
		e.Data = &TxEventData{}
	case EventTypePrimaryChange:
		e.Data = &PrimaryChangeEventData{}
	default:
		e.Data = nil
	}
	if err := json.Unmarshal(v.Data, &e.Data); err != nil {
		return err
	}
	return nil
}

type eventJSON struct {
	Type string          `json:"type"`
	DB   string          `json:"db,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

type InitEventData struct {
	IsPrimary bool   `json:"isPrimary"`
	Hostname  string `json:"hostname,omitempty"`
}

type TxEventData struct {
	TXID              string    `json:"txID"`              // ltx.TXID
	PostApplyChecksum string    `json:"postApplyChecksum"` // ltx.Checksum
	PageSize          uint32    `json:"pageSize"`
	Commit            uint32    `json:"commit"`
	Timestamp         time.Time `json:"timestamp"`
}

type PrimaryChangeEventData struct {
	IsPrimary bool   `json:"isPrimary"`
	Hostname  string `json:"hostname,omitempty"`
}
