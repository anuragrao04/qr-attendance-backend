package main

import (
	"github.com/anuragrao04/qr-attendance-backend/auth"
	"github.com/anuragrao04/qr-attendance-backend/database"
	"github.com/anuragrao04/qr-attendance-backend/handlers"
	"github.com/gin-gonic/gin"
)

func main() {

	// database
	database.Connect()
	database.ConnectGORM()

	// webauthn
	auth.Init()

	// router
	router := gin.Default()
	router.GET("/create-attendance-session", handlers.CreateSession)
	router.GET("/scan-qr", handlers.StudentScan)

	router.POST("/auth/register/begin", auth.BeginRegistration)
	router.POST("/auth/register/finish", auth.FinishRegistration)

	router.POST("/auth/login/begin", auth.BeginLogin)
	router.POST("/auth/login/finish", auth.FinishLogin)

	router.GET("/auth/check-if-registered-from-cookie", auth.CheckIfRegisteredCookie)
	router.GET("/auth/check-if-registered-from-header", auth.CheckIfRegisteredHeader)

	router.Run(":6969")
}
