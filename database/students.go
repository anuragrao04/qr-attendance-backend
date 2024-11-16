package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/anuragrao04/qr-attendance-backend/models"
)

func GetStudentsInAClassroom(studentTableName string) (students []models.StudentInASession, err error) {
	DBMutex.Lock()
	defer DBMutex.Unlock()
	var rows *sql.Rows
	rows, err = DB.Query(fmt.Sprintf("SELECT srn, prn, name FROM %s", studentTableName))
	if err != nil {
		log.Println("Failed to query students in classroom:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var s models.StudentInASession
		err = rows.Scan(&s.SRN, &s.PRN, &s.Name)
		if err != nil {
			log.Println("Failed to scan student:", err)
			return
		}
		s.IsPresent = false // Initialize to false
		students = append(students, s)
	}
	return
}
