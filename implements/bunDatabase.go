package implements

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"github.com/uptrace/bun"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type BunDatabase struct {
	db *bun.DB
}

func NewBunDatabase(db *bun.DB) *BunDatabase {
	return &BunDatabase{db: db}
}

func (self *BunDatabase) submissionQueryHook(
	ctx context.Context,
	subId int64,
) error {
	_, err := self.db.NewUpdate().Model((*models.Job)(nil)).
		Set("status = ?", models.StatusTimeout).
		Where("subid = ?", subId).
		Where("deadline < ?", time.Now().Unix()).
		Where("status = ?", models.StatusRunning).Exec(ctx)
	return err
}

func (self *BunDatabase) GetLastSubmission(
	ctx context.Context,
	csId string,
	problemId int64,
) (models.Submission, error) {
	var ret models.Submission
	err := self.db.NewSelect().Model(&ret).
		Where("csid = ?", csId).
		Where("problem_id = ?", problemId).
		Order("id DESC").Limit(1).Scan(ctx)
	if err != nil {
		return ret, fmt.Errorf("select submission failed, %w", err)
	}
	if err = self.submissionQueryHook(ctx, ret.Id); err != nil {
		return ret, fmt.Errorf("submission query hook failed, %w", err)
	}
	err = self.db.NewSelect().Model(&ret).
		WherePK().Relation("Jobs").Scan(ctx)
	if err != nil {
		return ret, fmt.Errorf("select submission failed, %w", err)
	}
	return ret, nil
}

func (self *BunDatabase) GetSubmission(
	ctx context.Context,
	subId int64,
) (models.Submission, error) {
	ret := models.Submission{Id: subId}
	if err := self.submissionQueryHook(ctx, subId); err != nil {
		return ret, fmt.Errorf("submission query hook failed, %w", err)
	}
	err := self.db.NewSelect().Model(&ret).
		WherePK().Relation("Jobs").Scan(ctx)
	return ret, err
}

func (self *BunDatabase) SkipJobs(
	ctx context.Context,
	subId int64,
) error {
	_, err := self.db.NewUpdate().Model((*models.Job)(nil)).
		Set("status = ?", models.StatusSkip).
		Where("subid = ?", subId).
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return q.
				Where("status = ?", models.StatusBlocked).
				WhereOr("status = ?", models.StatusPending)
		}).
		Exec(ctx)
	return err
}

func (self *BunDatabase) UpdateJob(
	ctx context.Context,
	job *models.Job,
) error {
	_, err := self.db.NewUpdate().Model(job).WherePK().Exec(ctx)
	return err
}

func (self *BunDatabase) InsertSubmissionAndJobs(
	ctx context.Context,
	submission *models.Submission,
	problem models.Problem,
) error {
	err := self.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := tx.NewInsert().Model(submission).Scan(ctx)
		if err != nil {
			return fmt.Errorf("submission insertion failed, %w", err)
		}

		for stageName, stage := range problem.Stages {
			for _, testcase := range stage.Testcases {
				job := models.Job{
					SubId:     submission.Id,
					StageName: stageName,
					TestName:  testcase.Name,

					Status: models.StatusBlocked,
				}

				_, err = tx.NewInsert().Model(&job).Exec(ctx)
				if err != nil {
					return fmt.Errorf("job insertion failed, %w", err)
				}
			}
		}

		_, err = tx.NewUpdate().Model(submission).WherePK().Exec(ctx)
		if err != nil {
			return fmt.Errorf("submission update failed, %w", err)
		}

		return nil
	})
	return err
}

func (self *BunDatabase) GetUserResource(
	ctx context.Context,
	csId string,
) (models.UserResource, error) {
	var ret models.UserResource
	err := self.db.NewSelect().Model(&ret).
		Where("csid = ?", csId).Scan(ctx)
	return ret, err
}

func (self *BunDatabase) IncreaseScore(
	ctx context.Context,
	subId int64,
	stageName string,
	delta int64,
) error {
	_, err := self.db.NewUpdate().Model((*models.Submission)(nil)).
		Set("score_score = score_score + ?", delta).
		Where("id = ?", subId).Exec(ctx)
	if err != nil {
		return fmt.Errorf("db score update failed, %w", err)
	}
	_, err = self.db.NewUpdate().Model((*models.Submission)(nil)).
		Set("score_stages = score_stages || ?",
			map[string]int64{stageName: delta}).
		Where("id = ?", subId).Exec(ctx)
	if err != nil {
		return fmt.Errorf("db stage score update failed, %w", err)
	}
	return nil
}

func (self *BunDatabase) GetScoresWithStages(
	ctx context.Context,
	problemId int64,
) ([]models.UserScoreWithStages, error) {
	ret := []models.UserScoreWithStages{}
	var submissions []models.Submission
	err := self.db.NewSelect().DistinctOn("csid").Model(&submissions).
		Column("csid", "score_score", "score_stages").
		Where("problem_id = ?", problemId).
		OrderExpr("csid, id DESC").
		Scan(ctx)
	if err != nil {
		return ret, fmt.Errorf("score query failed, %w", err)
	}

	for _, submission := range submissions {
		userScore := models.UserScoreWithStages{
			CsId:  submission.CsId,
			Score: submission.Score,
		}
		ret = append(ret, userScore)
	}

	return ret, nil
}

func (self *BunDatabase) GetScores(
	ctx context.Context,
	problemId int64,
) ([]models.UserScore, error) {
	ret := []models.UserScore{}
	var submissions []models.Submission
	err := self.db.NewSelect().DistinctOn("csid").Model(&submissions).
		Column("csid", "score_score").
		Where("problem_id = ?", problemId).
		OrderExpr("csid, id DESC").
		Scan(ctx)
	if err != nil {
		return ret, fmt.Errorf("score query failed, %w", err)
	}

	for _, submission := range submissions {
		userScore := models.UserScore{
			CsId:  submission.CsId,
			Score: submission.Score.Score,
		}
		ret = append(ret, userScore)
	}

	return ret, nil
}

func (self *BunDatabase) InsertUserResources(
	ctx context.Context,
	userResources []models.UserResource,
) error {
	for _, newUser := range userResources {
		err := self.db.RunInTx(ctx, nil,
			func(ctx context.Context, tx bun.Tx) error {
				var oldUser models.UserResource
				err := tx.NewSelect().
					Model(&oldUser).
					Where("csid = ?", newUser.CsId).
					Scan(ctx)
				if err == sql.ErrNoRows {
					// Insert New User
					_, err := tx.NewInsert().
						Model(&newUser).
						Exec(ctx)
					if err != nil {
						return fmt.Errorf("insert user failed, %w", err)
					}
					return nil
				}
				if err != nil {
					return fmt.Errorf("select user failed, %w", err)
				}

				// TODO: JSON Merge
				b, err := utils.JSONMerge(oldUser, newUser)
				if err != nil {
					return fmt.Errorf("merge user failed, %w", err)
				}

				var mergedUser models.UserResource
				if err := json.Unmarshal(b, &mergedUser); err != nil {
					return fmt.Errorf("json unmarshal failed, %w", err)
				}

				_, err = tx.NewUpdate().
					Model((*models.UserResource)(nil)).
					Set("args = ?", mergedUser.Args).
					Where("csid = ?", newUser.CsId).
					Exec(ctx)
				if err != nil {
					return fmt.Errorf("update user failed, %w", err)
				}

				return nil
			})
		if err != nil {
			return fmt.Errorf("insert user resource '%+v' failed, %w", newUser, err)
		}
	}
	return nil
}

func (self *BunDatabase) ExistUserResource(
	ctx context.Context,
	csId string,
) (bool, error) {
	return self.db.NewSelect().
		Model((*models.UserResource)(nil)).
		Where("csid = ?", csId).
		Exists(ctx)
}

func (self *BunDatabase) UpsertUser(
	ctx context.Context,
	user models.UserResource,
) error {
	err := self.db.NewInsert().
		Model(&user).
		On("CONFLICT (args) DO UPDATE").
		Set("args = EXCLUDED.args").
		Scan(ctx)
	return err
}

func (self *BunDatabase) GetSubmissionsAndCount(
	ctx context.Context,
	csId string,
	isCsId bool,
	pageNum int,
	isPageNum bool,
	pageSize int,
	problemId int64,
	isProblemId bool,
) ([]models.Submission, int, error) {
	query := self.db.NewSelect().Order("id DESC").Limit(pageSize)
	if isCsId {
		query = query.Where("csid = ?", csId)
	}
	if isPageNum {
		query = query.Offset(pageNum * pageSize)
	}
	if isProblemId {
		query = query.Where("problem_id = ?", problemId)
	}

	var ret []models.Submission
	count, err := query.Model(&ret).ScanAndCount(ctx)
	return ret, count, err
}

func (self *BunDatabase) GetUsers(
	ctx context.Context,
) ([]models.UserResource, error) {
	var ret []models.UserResource
	err := self.db.NewSelect().Model(&ret).Scan(ctx)
	return ret, err
}

func (self *BunDatabase) GetSession(
	ctx context.Context,
	csId string,
) (models.Session, error) {
	ret := models.Session{CsId: csId}
	err := self.db.NewSelect().Model(&ret).WherePK().Scan(ctx)
	return ret, err
}

func (self *BunDatabase) UpsertSession(
	ctx context.Context,
	session *models.Session,
) error {
	_, err := self.db.NewInsert().Model(session).
		On("CONFLICT (cs_id) DO UPDATE").
		Set("cs_id = EXCLUDED.cs_id").
		Set("groups = EXCLUDED.groups").
		Set("refresh_token = EXCLUDED.refresh_token").
		Set("is_valid = EXCLUDED.is_valid").
		Exec(ctx)
	return err
}

func (self *BunDatabase) UpdateRefreshToken(
	ctx context.Context,
	csId string,
) (models.Session, error) {
	ret := models.Session{
		CsId: csId,
	}
	err := self.db.NewSelect().Model(&ret).WherePK().Scan(ctx)
	if err != nil {
		return ret, err
	}
	if err := ret.RotateRefreshToken(); err != nil {
		return ret, err
	}
	err = self.db.NewUpdate().Model((*models.Session)(nil)).
		Where("cs_id = ?", csId).
		Set("refresh_token = ?", ret.RefreshToken).
		Returning("*").Scan(ctx, &ret)
	return ret, nil
}

func (self *BunDatabase) InvalidateSession(
	ctx context.Context,
	csId string,
) error {
	_, err := self.db.NewUpdate().Model((*models.Session)(nil)).
		Where("cs_id = ?", csId).
		Set("is_valid = ?", false).
		Exec(ctx)
	return err
}

func (self *BunDatabase) getProblems(
	ctx context.Context,
	isTool bool,
) (map[string]models.Problem, error) {
	ret := make(map[string]models.Problem)
	var problemsMarshaled []models.ProblemMarshaled
	err := self.db.NewSelect().
		Model(&problemsMarshaled).
		OrderExpr("problem_id, sha1").
		DistinctOn("problem_id").
		Where("is_tool = ?", isTool).
		Where("is_enabled = ?", true).
		Scan(ctx)
	if err != nil {
		return ret, err
	}

	for _, problem := range problemsMarshaled {
		ret[problem.SHA1], err = problem.Unmarshal()
		if err != nil {
			return ret, err
		}
	}
	return ret, nil
}

func (self *BunDatabase) GetProblems(
	ctx context.Context,
) (map[string]models.Problem, error) {
	return self.getProblems(ctx, false)
}

func (self *BunDatabase) GetTools(
	ctx context.Context,
) (map[string]models.Tool, error) {
	return self.getProblems(ctx, true)
}

func (self *BunDatabase) getProblem(
	ctx context.Context,
	sha1 string,
	isTool bool,
) (models.Problem, error) {
	var problemMarshaled models.ProblemMarshaled
	err := self.db.NewSelect().
		Model(&problemMarshaled).
		Where("sha1 = ?", sha1).
		Where("is_tool = ?", isTool).
		Scan(ctx)
	if err != nil {
		return models.Problem{}, err
	}

	return problemMarshaled.Unmarshal()
}

func (self *BunDatabase) GetProblem(
	ctx context.Context,
	sha1 string,
) (models.Problem, error) {
	return self.getProblem(ctx, sha1, false)
}

func (self *BunDatabase) GetTool(
	ctx context.Context,
	sha1 string,
) (models.Problem, error) {
	return self.getProblem(ctx, sha1, true)
}

func (self *BunDatabase) GetProblemOrTool(
	ctx context.Context,
	sha1 string,
) (models.Problem, error) {
	var problemMarshaled models.ProblemMarshaled
	err := self.db.NewSelect().
		Model(&problemMarshaled).
		Where("sha1 = ?", sha1).
		Scan(ctx)
	if err != nil {
		return models.Problem{}, err
	}

	return problemMarshaled.Unmarshal()
}

// get problem without check whether it is a tool
func (self *BunDatabase) getProblemById(
	ctx context.Context,
	problemId models.ProblemIdType,
	isTool bool,
) (models.Problem, error) {
	var problemMarshaled models.ProblemMarshaled
	err := self.db.NewSelect().Model(&problemMarshaled).
		Where("problem_id = ?", problemId).
		Where("is_tool = ?", isTool).
		Where("is_enabled = ?", true).
		Scan(ctx)
	if err != nil {
		return models.Problem{}, err
	}

	return problemMarshaled.Unmarshal()
}

func (self *BunDatabase) GetProblemById(
	ctx context.Context,
	problemId models.ProblemIdType,
) (models.Problem, error) {
	return self.getProblemById(ctx, problemId, false)
}

func (self *BunDatabase) GetToolById(
	ctx context.Context,
	problemId models.ProblemIdType,
) (models.Problem, error) {
	return self.getProblemById(ctx, problemId, true)
}

func (self *BunDatabase) GetProblemOrToolById(
	ctx context.Context,
	problemId models.ProblemIdType,
) (models.Problem, error) {
	var problemMarshaled models.ProblemMarshaled
	err := self.db.NewSelect().Model(&problemMarshaled).
		Where("problem_id = ?", problemId).
		Where("is_enabled = ?", true).
		Scan(ctx)
	if err != nil {
		return models.Problem{}, err
	}

	return problemMarshaled.Unmarshal()
}

func (self *BunDatabase) getAllProblems(
	ctx context.Context,
	isTool bool,
) (map[string]models.Problem, error) {
	ret := make(map[string]models.Problem)
	var problemsMarshaled []models.ProblemMarshaled
	err := self.db.NewSelect().
		Model(&problemsMarshaled).
		Where("is_tool = ?", isTool).
		Scan(ctx)
	if err != nil {
		return ret, err
	}

	for _, problem := range problemsMarshaled {
		ret[problem.SHA1], err = problem.Unmarshal()
		if err != nil {
			return ret, err
		}
	}
	return ret, nil
}

func (self *BunDatabase) GetAllProblems(
	ctx context.Context,
) (map[string]models.Problem, error) {
	return self.getAllProblems(ctx, false)
}

func (self *BunDatabase) GetAllTools(
	ctx context.Context,
) (map[string]models.Tool, error) {
	return self.getAllProblems(ctx, true)
}

func (self *BunDatabase) InsertProblem(
	ctx context.Context,
	problem models.Problem,
) error {
	problemMarshaled, err := problem.Marshal()
	if err != nil {
		return err
	}

	_, err = self.db.NewInsert().
		Model(&problemMarshaled).
		On("CONFLICT (sha1) DO NOTHING").
		Exec(ctx)
	return err
}

func (self *BunDatabase) InsertContentFiles(
	ctx context.Context,
	contentFiles []models.ContentFile,
) error {
	if contentFiles == nil {
		return nil
	}
	_, err := self.db.NewInsert().
		Model(&contentFiles).
		On("CONFLICT (sha1) DO NOTHING").
		Exec(ctx)
	return err
}

func (self *BunDatabase) GetContentFiles(
	ctx context.Context,
	sha1s []string,
) ([]models.ContentFile, error) {
	var res []models.ContentFile
	err := self.db.NewSelect().
		Model(&res).
		Where("sha1 IN (?)", bun.In(sha1s)).
		Scan(ctx)
	if err != nil {
		return []models.ContentFile{},
			fmt.Errorf("content files select failed, %w", err)
	}

	var ret []models.ContentFile
	for _, contentFile := range res {
		if contentFile.IsTemplate {
			contentFile.Template, err = template.New(contentFile.Name).
				Parse(contentFile.Payload)
			if err != nil {
				return []models.ContentFile{},
					fmt.Errorf("templating failed, %w", err)
			}
		}
		ret = append(ret, contentFile)
	}
	return ret, nil
}

func (self *BunDatabase) GetAllContentFiles(
	ctx context.Context,
) ([]models.ContentFile, error) {
	var ret []models.ContentFile
	err := self.db.NewSelect().
		Model(&ret).
		Scan(ctx)
	if err != nil {
		return []models.ContentFile{},
			fmt.Errorf("content files select failed, %w", err)
	}

	for i, contentFile := range ret {
		if contentFile.IsTemplate {
			ret[i].Template, err = template.New(contentFile.Name).
				Parse(contentFile.Payload)
			if err != nil {
				return []models.ContentFile{},
					fmt.Errorf("templating failed, %w", err)
			}
		}
	}
	return ret, nil
}

func (self *BunDatabase) InsertTool(
	ctx context.Context,
	tool models.Tool,
) error {
	return self.InsertProblem(ctx, tool)
}

func (self *BunDatabase) UpdateProblem(
	ctx context.Context,
	sha1 string,
	enable bool,
) error {
	if enable {
		return self.db.RunInTx(ctx, &sql.TxOptions{},
			func(ctx context.Context, tx bun.Tx) error {
				var problem models.ProblemMarshaled
				err := tx.NewSelect().
					Model(&problem).
					Where("sha1 = ?", sha1).
					Scan(ctx)
				if err != nil {
					return fmt.Errorf("problem select failed, %w", err)
				}
				_, err = tx.NewUpdate().
					Model((*models.ProblemMarshaled)(nil)).
					Set("is_enabled = CASE WHEN sha1 = ? THEN TRUE ELSE FALSE END", sha1).
					Where("problem_id = ?", problem.ProblemId).
					Exec(ctx)
				if err != nil {
					return fmt.Errorf("problem update failed, %w", err)
				}
				return nil
			})
	} else {
		_, err := self.db.NewUpdate().
			Model((*models.ProblemMarshaled)(nil)).
			Where("sha1 = ?", sha1).
			Set("is_enabled = ?", false).
			Exec(ctx)
		return err
	}
}

func (self *BunDatabase) UpdateTool(
	ctx context.Context,
	sha1 string,
	enable bool,
) error {
	return self.UpdateProblem(ctx, sha1, enable)
}

func (self *BunDatabase) HydrateContentFiles(
	ctx context.Context,
	contentFiles *[]models.ContentFile,
) error {
	var sha1s []string
	for _, file := range *contentFiles {
		sha1s = append(sha1s, file.SHA1)
	}

	res, err := self.GetContentFiles(ctx, sha1s)
	if err != nil {
		return fmt.Errorf("get content files failed, %w", err)
	}
	// fmt.Fprintf(os.Stderr, "[DEBUG] %+##v\n", res)
	*contentFiles = res
	return nil
}

func (self *BunDatabase) HydrateProblem(
	ctx context.Context,
	problem *models.Problem,
) error {
	if err := self.HydrateContentFiles(
		ctx, &problem.PreScript.Files); err != nil {
		return fmt.Errorf("hydrate problem prescript failed, %w", err)
	}
	if err := self.HydrateContentFiles(
		ctx, &problem.PostScript.Files); err != nil {
		return fmt.Errorf("hydrate problem postscript failed, %w", err)
	}

	for stageName, stage := range problem.Stages {
		if err := self.HydrateContentFiles(
			ctx, &stage.PreScript.Files); err != nil {
			return fmt.Errorf("hydrate stage prescript failed, %w", err)
		}
		if err := self.HydrateContentFiles(
			ctx, &stage.PostScript.Files); err != nil {
			return fmt.Errorf("hydrate stage postscript failed, %w", err)
		}

		for i := range stage.Testcases {
			if err := self.HydrateContentFiles(ctx,
				&stage.Testcases[i].Content.Files); err != nil {
				return fmt.Errorf("hydrate content files failed, %w", err)
			}
		}
		problem.Stages[stageName] = stage
	}
	return nil
}

func (self *BunDatabase) EnableAllProblem(
	ctx context.Context,
) error {
	_, err := self.db.NewUpdate().
		Model((*models.ProblemMarshaled)(nil)).
		Set("is_enabled = ?", true).
		Where("is_enabled = ?", false).
		Exec(ctx)
	return err
}

func (self *BunDatabase) DeleteProblemAndSubmissions(
	ctx context.Context,
	sha1 string,
) error {
	return self.db.RunInTx(ctx, &sql.TxOptions{},
		func(ctx context.Context, tx bun.Tx) error {
			_, err := tx.NewDelete().
				Model((*models.ProblemMarshaled)(nil)).
				Where("sha1 = ?", sha1).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("delete problem failed, %w", err)
			}
			_, err = tx.NewDelete().
				Model((*models.Submission)(nil)).
				Where("problem_sha1 = ?", sha1).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("delete submissions failed, %w", err)
			}
			return nil
		})
}

func (self *BunDatabase) DeleteSubmission(
	ctx context.Context,
	subId int64,
) error {
	_, err := self.db.NewDelete().
		Model((*models.Submission)(nil)).
		Where("id = ?", subId).
		Exec(ctx)
	return err
}

func (self *BunDatabase) DeleteUser(
	ctx context.Context,
	csId string,
) error {
	_, err := self.db.NewDelete().
		Model((*models.UserResource)(nil)).
		Where("csid = ?", csId).
		Exec(ctx)
	return err
}
