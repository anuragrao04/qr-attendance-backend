package models

type Session struct {
	CurrentRandomID           RandomID
	PastRandomIDs             []RandomID
	ClassroomTable            string
	Students                  []StudentInASession
	TeacherQRRenderingLatency int64
}

type StudentInASession struct {
	PRN       string `json:"PRN"`
	SRN       string `json:"SRN"`
	Name      string `json:"name"`
	IsPresent bool   `json:"isPresent"`
}

type RandomID struct {
	ID        uint32
	CreatedAt int64
	ExpiredAt int64
}
