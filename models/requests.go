package models

type ScanMessage struct {
	SessionID       uint32 `json:"sessionID"`
	ScannedRandomID uint32 `json:"scannedRandomID"`
	ScannedAt       string `json:"scannedAt"` // this is later parsed to uint64. This is a string to avoid overflow
	SRN             string `json:"SRN"`
}
