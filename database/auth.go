package database

import (
	"bytes"
	"errors"

	"github.com/anuragrao04/qr-attendance-backend/models"
	"github.com/go-webauthn/webauthn/webauthn"
)

func CreateUser(SRN string) (models.User, error) {
	GORMDBMutex.Lock()
	defer GORMDBMutex.Unlock()
	user := models.User{
		SRN: SRN,
	}
	err := GORMDB.Create(&user).Error
	if err != nil {
		return user, err
	}
	return user, nil
}

func GetUser(SRN string) (models.User, error) {
	GORMDBMutex.Lock()
	defer GORMDBMutex.Unlock()
	var user models.User
	err := GORMDB.Where("SRN = ?", SRN).First(&user).Error
	if err != nil {
		return user, err
	}
	return user, nil
}

func AddCredential(SRN string, credential *webauthn.Credential) error {

	// Get the user from the database
	user, err := GetUser(SRN)
	if err != nil {
		return err
	}
	GORMDBMutex.Lock()
	defer GORMDBMutex.Unlock()

	// Append the new credential to the existing slice
	user.Credentials = append(user.Credentials, *credential)

	// Save the updated user record
	return GORMDB.Save(&user).Error
}

func UpdateCredential(SRN string, credential *webauthn.Credential) error {

	// Get the user from the database
	user, err := GetUser(SRN)
	if err != nil {
		return err
	}

	GORMDBMutex.Lock()
	defer GORMDBMutex.Unlock()
	// Find and update the matching credential
	updated := false
	for i, existingCredential := range user.Credentials {
		if bytes.Equal(existingCredential.ID, credential.ID) {
			user.Credentials[i] = *credential
			updated = true
			break
		}
	}

	if !updated {
		return errors.New("credential not found")
	}

	// Save the updated user record
	return GORMDB.Save(&user).Error
}
