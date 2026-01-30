package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

// GetHealthz
//
//	@success	200
//	@router		/api/healthz [get]
func (self *Handlers) GetHealthz(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	return utils.Success(w)
}
