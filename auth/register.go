package auth

import (
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/anuragrao04/qr-attendance-backend/database"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"
)

var WebAuthnRegisterSessions sync.Map

func BeginRegistration(c *gin.Context) {
	SRN := c.GetHeader("SRN")
	user, err := database.GetUser(SRN)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user, err = database.CreateUser(SRN)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	if len(user.Credentials) > 0 {
		// this guy is already registered with another authenticator
		c.JSON(http.StatusBadRequest, gin.H{"error": "User already registered"})
		return
	}

	options, session, err := WebAuthn.BeginRegistration(user)

	WebAuthnRegisterSessions.Store(SRN, session)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, options)
}

func FinishRegistration(c *gin.Context) {
	SRN := c.GetHeader("SRN")
	log.Println("Getting session")
	session_untyped, ok := WebAuthnRegisterSessions.Load(SRN)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session not found"})
		return
	}
	session := session_untyped.(*webauthn.SessionData)
	log.Println("Got session")
	user, err := database.GetUser(SRN)
	log.Println("Got user")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	credential, err := WebAuthn.FinishRegistration(user, *session, c.Request)
	log.Println("finished webauthn lib stuff")

	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Println("Trying to add credentials")
	if err := database.AddCredential(user.SRN, credential); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Println("Trying to delete session")
	WebAuthnRegisterSessions.Delete(SRN)
	log.Println("Deleted session")

	c.SetCookie(
		"SRN",            // Cookie name
		SRN,              // Cookie value
		int(^uint(0)>>1), // Max age in seconds (big value)
		"/",              // Path
		"",               // Domain (default: current domain)
		true,             // Secure (true to allow only over HTTPS)
		true,             // HttpOnly (true to disallow JavaScript access)
	)
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

func CheckIfRegisteredCookie(c *gin.Context) {
	SRN, err := c.Cookie("SRN")
	if err != nil {
		// cookie was not there, aka not registered
		c.JSON(http.StatusOK, gin.H{"registered": false})
		return
	}
	user, err := database.GetUser(SRN)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"registered": false})
			return
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	if len(user.Credentials) > 0 {
		c.JSON(http.StatusOK, gin.H{"registered": true})
		return
	} else {
		c.JSON(http.StatusOK, gin.H{"registered": false})
		return
	}
}

func CheckIfRegisteredHeader(c *gin.Context) {
	SRN := c.GetHeader("SRN")
	if SRN == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SRN header not found"})
		return
	}
	user, err := database.GetUser(SRN)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"registered": false})
			return
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	if len(user.Credentials) > 0 {
		c.JSON(http.StatusOK, gin.H{"registered": true})
		return
	} else {
		c.JSON(http.StatusOK, gin.H{"registered": false})
		return
	}
}
