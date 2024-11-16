package models

type Session struct {
	CurrentRandomID RandomID
	PastRandomIDs   []RandomID
	ClassroomTable  string
	Students        []StudentInASession
}

type StudentInASession struct {
	PRN       string
	SRN       string
	Name      string
	IsPresent bool
}

type RandomID struct {
	ID        uint32
	CreatedAt int64
	ExpiredAt int64
}
