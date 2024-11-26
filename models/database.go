package models

import (
	"bytes"
	"database/sql/driver"
	"encoding/gob"
	"errors"

	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	SRN         string             `json:"SRN" gorm:"index"`
	Credentials CredentialsWrapper `gorm:"type:blob"`
}

// CredentialsWrapper wraps []webauthn.Credential for GORM
type CredentialsWrapper []webauthn.Credential

// Value serializes CredentialsWrapper into a binary format
func (cw CredentialsWrapper) Value() (driver.Value, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(cw)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Scan deserializes a binary format into CredentialsWrapper
func (cw *CredentialsWrapper) Scan(value interface{}) error {
	bytesData, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	reader := bytes.NewReader(bytesData)
	dec := gob.NewDecoder(reader)
	return dec.Decode(cw)
}

// WebAuthnID returns the SRN as a byte slice
func (u User) WebAuthnID() []byte {
	return []byte(u.SRN)
}

// WebAuthnName returns the SRN as the user's name
func (u User) WebAuthnName() string {
	return u.SRN
}

// WebAuthnDisplayName returns the SRN as the display name
func (u User) WebAuthnDisplayName() string {
	return u.SRN
}

// WebAuthnCredentials returns the user's credentials
func (u User) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}
