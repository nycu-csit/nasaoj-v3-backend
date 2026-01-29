package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

// GetLogin
//
//	@Param			source	query	string	false	"Login source"	attribute(optional)
//	@Description	OAuth login endpoint
//	@Header			302	{}	string	"AuthCodeURL(state)"
//	@Response		302	{}	string	"302 Found"
//	@Router			/auth/login [get]
func (self *Handlers) GetLogin(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	var expire = time.Now().Add(365 * 24 * time.Hour)

	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name:    "oauthstate",
		Value:   state,
		Expires: expire,
	})

	if req.URL.Query().Get("source") == "cli" {
		w.Header().Set("Location", self.cliOAuth2Config.AuthCodeURL(state))
		return utils.Success(w)
	} else {
		return utils.Redirect(w, self.webOAuth2Config.AuthCodeURL(state))
	}
}
