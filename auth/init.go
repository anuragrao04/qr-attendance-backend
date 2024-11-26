package auth

import (
	"github.com/go-webauthn/webauthn/webauthn"
)

var WebAuthn *webauthn.WebAuthn

func Init() {
	wconfig := &webauthn.Config{
		RPDisplayName: "QR Attendance", // Display Name for your site
		// RPID:          "attendance.anuragrao.site",                                            // Generally the FQDN for your site
		RPID:      "localhost",
		RPOrigins: []string{"http://localhost:3000", "https://attendance.anuragrao.site"}, // The origin URLs allowed for WebAuthn requests
	}
	var err error
	if WebAuthn, err = webauthn.New(wconfig); err != nil {
		panic(err)
	}
}
