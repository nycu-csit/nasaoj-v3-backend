package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type PutNamespaceUsersPlainInputUser struct {
	Username string            `json:"username" yaml:"username"`
	Password string            `json:"password" yaml:"password"`
	Args     map[string]string `json:"args" yaml:"args"`
	// IsAdmin grants namespace admin of the target namespace.
	IsAdmin bool `json:"isAdmin" yaml:"isAdmin"`
}

type PutNamespaceUsersCSSSOInputUser struct {
	CsId string            `json:"csId" yaml:"csId"`
	Args map[string]string `json:"args" yaml:"args"`
	// IsAdmin grants namespace admin of the target namespace.
	IsAdmin bool `json:"isAdmin" yaml:"isAdmin"`
}

// PutNamespaceUsersPlain
//
//	@Summary		Import plain-login users into a namespace
//	@Description	Creates the users whose username is new and makes every
//	@Description	user a member of the namespace from the X-Namespace header.
//	@Description	Existing users are only added to the namespace: their
//	@Description	password, args and global admin flag are left untouched.
//	@Description	isAdmin grants namespace admin. Global admin or admin of
//	@Description	that namespace.
//	@Accept			json
//	@Produce		json
//	@Param			X-Namespace	header		string								true	"namespace name or id"
//	@Param			request		body		[]PutNamespaceUsersPlainInputUser	true	"users"
//	@Success		200			{object}	[]models.NamespaceUserImported
//	@Failure		400			{object}	string
//	@Failure		403			{object}	string
//	@Router			/api/admin/namespace/users/plain [put]
func (self *Handlers) PutNamespaceUsersPlain(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	var inputs []PutNamespaceUsersPlainInputUser
	if err := readJSON(req, &inputs); err != nil {
		return err
	}

	users := make([]models.NamespaceUserImport, len(inputs))
	for i, in := range inputs {
		if in.Username == "" || in.Password == "" {
			return middlewares.HTTPError{
				StatusCode: http.StatusBadRequest,
				Message:    "username and password are required",
			}
		}
		users[i] = models.NamespaceUserImport{
			User: models.UserResource{
				Username: in.Username,
				Args:     in.Args,
				Source:   models.LoginSourcePlain,
			},
			Cred:    &models.PlainCred{Passhash: utils.HashPass(in.Password)},
			IsAdmin: in.IsAdmin,
		}
	}
	return self.importNamespaceUsers(w, req, users)
}

// PutNamespaceUsersCSSSO
//
//	@Summary		Import CSSSO users into a namespace
//	@Description	Creates the users whose CS id is new and makes every user a
//	@Description	member of the namespace from the X-Namespace header.
//	@Description	Existing users are only added to the namespace: their args
//	@Description	and global admin flag are left untouched. isAdmin grants
//	@Description	namespace admin. Global admin or admin of that namespace.
//	@Accept			json
//	@Produce		json
//	@Param			X-Namespace	header		string								true	"namespace name or id"
//	@Param			request		body		[]PutNamespaceUsersCSSSOInputUser	true	"users"
//	@Success		200			{object}	[]models.NamespaceUserImported
//	@Failure		400			{object}	string
//	@Failure		403			{object}	string
//	@Router			/api/admin/namespace/users/cssso [put]
func (self *Handlers) PutNamespaceUsersCSSSO(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	var inputs []PutNamespaceUsersCSSSOInputUser
	if err := readJSON(req, &inputs); err != nil {
		return err
	}

	users := make([]models.NamespaceUserImport, len(inputs))
	for i, in := range inputs {
		if in.CsId == "" {
			return middlewares.HTTPError{
				StatusCode: http.StatusBadRequest,
				Message:    "csId is required",
			}
		}
		users[i] = models.NamespaceUserImport{
			User: models.UserResource{
				Username: in.CsId,
				Args:     in.Args,
				Source:   models.LoginSourceCSSSO,
				SourceId: in.CsId,
			},
			IsAdmin: in.IsAdmin,
		}
	}
	return self.importNamespaceUsers(w, req, users)
}

func (self *Handlers) importNamespaceUsers(
	w http.ResponseWriter, req bunrouter.Request,
	users []models.NamespaceUserImport,
) error {
	ctx := req.Context()
	imported, err := self.namespaces.ImportNamespaceUsers(ctx,
		middlewares.NamespaceIdFromContext(ctx), users)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "failed to import namespace users",
			OriginError: err,
		}
	}
	return bunrouter.JSON(w, imported)
}

func readJSON(req bunrouter.Request, v any) error {
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "read payload failed",
			OriginError: err,
		}
	}
	if err := json.Unmarshal(b, v); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}
	return nil
}
