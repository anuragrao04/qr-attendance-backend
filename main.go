package main

import (
	"github.com/anuragrao04/qr-attendance-backend/database"
	"github.com/anuragrao04/qr-attendance-backend/handlers"
	"github.com/gin-gonic/gin"
)

func main() {

	// database
	database.Connect()
	database.ConnectGORM()
	router := gin.Default()
	router.GET("/create-attendance-session", handlers.CreateSession)
	router.GET("/scan-qr", handlers.StudentScan)
	router.Run(":6969")
}
