package api

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type PutContentFilesInput struct {
	SHA1       string `json:"sha1"`
	Name       string `json:"name"`
	Payload    string `json:"payload"`
	IsTemplate bool   `json:"isTemplate"`
}

// PutContentFiles
//
//	@Summary		Upload content files
//	@Description	Uploads SHA1-addressed content files used by problems and tools. Admin only.
//	@Param			request	body	[]PutContentFilesInput	true	"content files"
//	@Accept			json
//	@Success		200	{object}	string	"success"
//	@Failure		401	{object}	string
//	@Router			/api/admin/content-files [put]
func (self *Handlers) PutContentFiles(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	b, err := io.ReadAll(req.Body)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "read payload failed",
			OriginError: err,
		}
	}

	var inputs []PutContentFilesInput
	if err := json.Unmarshal(b, &inputs); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	var files []models.ContentFile
	for _, input := range inputs {
		// Content files are shared by every namespace and keyed by sha1, so
		// the key must really be the payload's hash - otherwise one namespace
		// admin could squat a sha1 another namespace will later upload, with
		// different content.
		h := sha1.Sum([]byte(input.Payload))
		if hex.EncodeToString(h[:]) != input.SHA1 {
			return middlewares.HTTPError{
				StatusCode: http.StatusBadRequest,
				Message:    "content file sha1 does not match payload",
			}
		}
		files = append(files, models.ContentFile{
			SHA1:       input.SHA1,
			Name:       input.Name,
			Payload:    input.Payload,
			IsTemplate: input.IsTemplate,
		})
	}

	if err := self.contents.InsertContentFiles(ctx, files); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "insert content files failed",
			OriginError: err,
		}
	}

	return utils.Success(w)
}
