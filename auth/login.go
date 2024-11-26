package auth

import (
	"net/http"
	"sync"

	"github.com/anuragrao04/qr-attendance-backend/database"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
)

var WebAuthnLoginSessions sync.Map

func BeginLogin(c *gin.Context) {
	SRN, err := c.Cookie("SRN")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SRN cookie not found"})
		return
	}
	user, err := database.GetUser(SRN)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(user.Credentials) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No credential found for this user"})
		return
	}
	options, session, err := WebAuthn.BeginLogin(user)
	WebAuthnLoginSessions.Store(SRN, session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, options)
}

func FinishLogin(c *gin.Context) {
	SRN, err := c.Cookie("SRN")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SRN cookie not found"})
		return
	}
	session_untyped, ok := WebAuthnLoginSessions.Load(SRN)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session not found"})
		return
	}
	session := session_untyped.(*webauthn.SessionData)
	user, err := database.GetUser(SRN)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	credential, err := WebAuthn.FinishLogin(user, *session, c.Request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = database.UpdateCredential(user.SRN, credential)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	WebAuthnLoginSessions.Delete(SRN)
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}
