package models

import "gorm.io/gorm"

type UserFingerprint struct {
	gorm.Model
	SRN                string `json:"SRN" gorm:"index"`
	BrowserFingerprint string `json:"browserFingerprint"`
}
