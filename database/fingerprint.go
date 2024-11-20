package database

import (
	"log"

	"github.com/anuragrao04/qr-attendance-backend/models"
)

func ValidateFingerprint(srn string, fingerprint string) (isFingerprintValid bool, err error) {
	GORMDBMutex.Lock()
	defer GORMDBMutex.Unlock()
	var student models.UserFingerprint
	result := GORMDB.First(&student, "srn = ?", srn)
	if result.Error != nil {
		if result.Error.Error() == "record not found" {

			// check if the fingerprint exists in the database (same fingerprint two SRNs)
			// disallow that

			result = GORMDB.First(&student, "fingerprint = ?", fingerprint)
			if result.Error == nil {
				// there was a student with the given fingerprint
				// kill him
				return false, nil
			}

			// create it
			log.Println("Creating new fingerprint for SRN", srn, ":", fingerprint)
			student = models.UserFingerprint{
				SRN:                srn,
				BrowserFingerprint: fingerprint,
			}
			result = GORMDB.Create(&student)
			if result.Error != nil {
				return false, result.Error
			}

			return true, nil

		} else {
			return false, result.Error
		}
	}

	if student.BrowserFingerprint != fingerprint {
		return false, nil
	}
	return true, nil
}
