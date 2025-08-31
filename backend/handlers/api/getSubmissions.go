package api

import (
	"net/http"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

type SubmissionsOutputSubmission struct {
	Id      int64          `json:"id"`
	RefSHA1 string         `json:"refSHA1"`
	RefId   int64          `json:"refId"`
	Type    models.RefType `json:"type"`

	CsId      string `json:"csId"`
	CreatedAt int64  `json:"createdAt"`
	Score     int64  `json:"score"`
}

type SubmissionsOutput struct {
	PageSize    int                           `json:"pageSize"`
	TotalRowCnt int                           `json:"totalRowCnt"`
	Submissions []SubmissionsOutputSubmission `json:"submissions"`
}

const pageSize int = 20

// GetAdminSubmissions
//
//	@Param		pageNum	query		int64				false	"Page number, one page is 20 submissions"	default(0)
//	@Param		csId	query		string				false	"Filter by CsId"							attribute(optional)
//	@Param		refId	query		int64				false	"Filter by RefId"							attribute(optional)
//	@Param		type	query		int					false	"Filter by Type"							attribute(optional)
//	@Success	200		{object}	SubmissionsOutput	"Submissions"
//	@Failure	400		{object}	string				"Problem Id invalid"
//	@Failure	401		{object}	string				"not admin or didn't login"
//	@Router		/api/admin/submissions [get]
func getAdminSubmissionsPlaceHolder() {}

// GetSubmissions
//
//	@Param		pageNum	query		int64				false	"Page number, one page is 20 submissions"	default(0)
//	@Param		refId	query		int64				false	"Filter by RefId"							attribute(optional)
//	@Param		type	query		int					false	"Filter by Type"							attribute(optional)
//	@Success	200		{object}	SubmissionsOutput	"Submissions"
//	@Failure	400		{object}	string				"Problem Id invalid"
//	@Failure	401		{object}	string				"didn't login"
//	@Router		/api/submissions [get]
func GetSubmissions(
	db interfaces.Database,
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		queryValues := req.URL.Query()
		var err error

		var csId string
		var isCsId bool = false
		if isAdmin {
			if queryValues.Has("csId") {
				csId = queryValues.Get("csId")
				isCsId = true
			}
		} else {
			privs, ok := req.Context().
				Value(types.Privileges("")).(models.Privileges)
			if !ok {
				return middlewares.HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "privs missing",
				}
			}
			csId = privs.OwnerCsId
			isCsId = true
		}

		var pageNum int
		var isPageNum bool = false
		if queryValues.Has("pageNum") {
			pageNum64, err := strconv.ParseInt(
				queryValues.Get("pageNum"), 10, 32)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "page num parse failed",
					OriginError: err,
				}
			}
			pageNum = int(pageNum64)
			isPageNum = true
		}

		var refType models.RefType
		var isRefType bool = false
		if queryValues.Has("type") {
			refType64, err := strconv.ParseInt(
				queryValues.Get("type"), 10, 32)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "failed to parse 'type'",
					OriginError: err,
				}
			}
			refType = int(refType64)
			isRefType = true
		}

		var refId int64
		var isRefId bool = false
		if queryValues.Has("refId") {
			refId, err = strconv.ParseInt(
				queryValues.Get("refId"), 10, 64)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "ref id parse failed",
					OriginError: err,
				}
			}
			isRefId = true
		}

		subs, count, err := db.GetSubmissionsAndCount(ctx,
			csId, isCsId,
			pageNum, isPageNum, pageSize,
			refId, isRefId,
			refType, isRefType)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "db query failed",
				OriginError: err,
			}
		}

		submissions := []SubmissionsOutputSubmission{}
		for _, sub := range subs {
			output := SubmissionsOutputSubmission{
				Id:      sub.Id,
				RefSHA1: sub.RefSHA1,
				RefId:   sub.RefId,
				Type:    sub.Type,

				CsId:      sub.CsId,
				CreatedAt: sub.CreatedAt,
				Score:     sub.Score.Score,
			}
			submissions = append(submissions, output)
		}
		return bunrouter.JSON(w, SubmissionsOutput{
			PageSize:    pageSize,
			TotalRowCnt: count,
			Submissions: submissions,
		})
	}
}
