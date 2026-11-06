package tests

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// This suite covers self-service user tokens (POST /api/token, and
// `judge admin gen token user` without a userId), username/password login
// in the CLI (--username), and the admin-flag bookkeeping around them:
// GET /api/admin/users reporting memberships, and a demoted global admin
// not keeping namespace admin of default.
//
// It reuses the "nsb" namespace set up by 21_namespace_test.go.

const (
	selfUserUser    = "self-user"
	selfUserPass    = "self-user-pass"
	selfNsAdminUser = "self-nsadmin"
	selfNsAdminPass = "self-nsadmin-pass"
	selfDemotedUser = "self-demoted"
	selfDemotedPass = "self-demoted-pass"
)

var (
	test28UserId    int64
	test28NsAdminId int64
	test28DemotedId int64
	test28NsBId     int64

	test28UserToken    string
	test28NsAdminToken string
)

// runJudgeIn runs the judge CLI with HOME pointed at home, so the CLI's
// session file and config never touch the real ones, feeding stdin (the
// password prompt reads its first line). It returns stdout, stderr and
// whether the process succeeded.
func runJudgeIn(
	t *testing.T, home, stdin string, args ...string,
) (string, string, bool) {
	t.Helper()

	cmd := exec.Command("judge", args...)
	cmd.Env = append(os.Environ(), "HOME="+home)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err == nil
}

// tokenFromOutput extracts the token printed by `judge admin gen token`.
func tokenFromOutput(out string) (string, bool) {
	_, rest, ok := strings.Cut(out, "Token: ")
	if !ok {
		return "", false
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 || fields[0] == "" {
		return "", false
	}
	return fields[0], true
}

// meOf returns /api/me as seen through the given bearer token.
func meOf(t *testing.T, token string) api.MeOutput {
	t.Helper()

	var me api.MeOutput
	resp, err := bearerReq(token).SetResult(&me).Get("/api/me")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("GET /api/me with token failed: %v %s", err, resp)
	}
	return me
}

func Test_28_1_Setup_users(t *testing.T) {
	stdin := `---
- username: '` + selfUserUser + `'
  password: '` + selfUserPass + `'
- username: '` + selfNsAdminUser + `'
  password: '` + selfNsAdminPass + `'
- username: '` + selfDemotedUser + `'
  password: '` + selfDemotedPass + `'
  isAdmin: true
`
	runJudgeStdin(t, stdin,
		"admin", "import", "users", "plain",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var users []models.UserResource
	resp, err := adminReq().SetResult(&users).Get("/api/admin/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("get users failed: %v %s", err, resp)
	}
	for _, u := range users {
		switch u.Username {
		case selfUserUser:
			test28UserId = u.Id
		case selfNsAdminUser:
			test28NsAdminId = u.Id
		case selfDemotedUser:
			test28DemotedId = u.Id
		}
	}
	if test28UserId == 0 || test28NsAdminId == 0 || test28DemotedId == 0 {
		t.Fatalf("self token test users not found (user=%d nsadmin=%d demoted=%d)",
			test28UserId, test28NsAdminId, test28DemotedId)
	}

	var namespaces []models.Namespace
	resp, err = adminReq().SetResult(&namespaces).Get("/api/admin/namespaces")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("get namespaces failed: %v %s", err, resp)
	}
	for _, ns := range namespaces {
		if ns.Name == nsB {
			test28NsBId = ns.Id
		}
	}
	if test28NsBId == 0 {
		t.Fatalf("namespace %q not found", nsB)
	}

	// self-nsadmin administers nsb only; it stays a plain member of default.
	resp, err = adminReq().
		SetHeader("X-Namespace", nsB).
		SetBody(api.PutNamespaceUsersInput{
			UserIds: []int64{test28NsAdminId},
			IsAdmin: true,
		}).
		Put("/api/admin/namespace/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("make self-nsadmin admin of nsb failed: %v %s", err, resp)
	}
}

func Test_28_2_Get_users_reports_memberships(t *testing.T) {
	var users []models.UserResource
	resp, err := adminReq().SetResult(&users).Get("/api/admin/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("get users failed: %v %s", err, resp)
	}

	for _, u := range users {
		if u.Id != test28NsAdminId {
			continue
		}
		if !slices.Contains(u.Namespaces, models.DefaultNamespaceId) ||
			!slices.Contains(u.Namespaces, test28NsBId) {
			t.Fatalf("expected self-nsadmin in default and nsb, got namespaces %v",
				u.Namespaces)
		}
		if !slices.Equal(u.AdminIn, []int64{test28NsBId}) {
			t.Fatalf("expected self-nsadmin adminIn [%d], got %v",
				test28NsBId, u.AdminIn)
		}
		return
	}
	t.Fatalf("self-nsadmin (%d) not in /api/admin/users", test28NsAdminId)
}

// Test_28_3_Demoted_admin_keeps_no_namespace_admin is the regression for
// default membership mirroring the global admin flag: after demotion the
// user must not still administer default.
func Test_28_3_Demoted_admin_keeps_no_namespace_admin(t *testing.T) {
	c := loginPlain(t, selfDemotedUser, selfDemotedPass)
	var me api.MeOutput
	resp, err := c.R().SetResult(&me).Get("/api/me")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("GET /api/me as admin failed: %v %s", err, resp)
	}
	if !me.IsAdmin {
		t.Fatal("expected self-demoted to start as a global admin")
	}
	if slices.Contains(me.AccessTokenClaim.Resource.AdminIn,
		models.DefaultNamespaceId) {
		t.Fatalf("a global admin's default membership must not carry "+
			"namespace admin, got adminIn %v",
			me.AccessTokenClaim.Resource.AdminIn)
	}

	// Demote.
	resp, err = adminReq().
		SetBody([]api.PutUsersPlainInputUser{{
			Username: selfDemotedUser,
			Password: selfDemotedPass,
			IsAdmin:  false,
		}}).
		Put("/api/admin/users/plain")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("demote self-demoted failed: %v %s", err, resp)
	}

	// A fresh session reflects the demotion: no admin page, no admin routes.
	c = loginPlain(t, selfDemotedUser, selfDemotedPass)
	me = api.MeOutput{}
	resp, err = c.R().SetResult(&me).Get("/api/me")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("GET /api/me after demotion failed: %v %s", err, resp)
	}
	if me.IsAdmin {
		t.Fatal("expected self-demoted to no longer be a global admin")
	}
	for _, ns := range me.Namespaces {
		if ns.IsAdmin {
			t.Fatalf("demoted user still administers namespace %q", ns.Name)
		}
	}

	resp, err = c.R().
		SetHeader("X-Namespace", "default").
		SetBody(api.AdminSubmitInput{
			SubmitInput: api.SubmitInput{Type: models.RefTypeProblem},
			AsOwnerId:   test28UserId,
		}).
		Post("/api/admin/submit")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("demoted user submitting as another user: expected 403, "+
			"got %d %s", resp.StatusCode(), resp.String())
	}
}

func Test_28_4_Self_token_validation(t *testing.T) {
	c := loginPlain(t, selfUserUser, selfUserPass)

	// Missing description.
	resp, err := c.R().
		SetBody(api.PostSelfTokenInput{Description: "   "}).
		Post("/api/token")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing description, got %d %s",
			resp.StatusCode(), resp.String())
	}

	// A static admin token belongs to no user: nothing to issue a token for.
	resp, err = adminReq().
		SetBody(api.PostSelfTokenInput{Description: "admin self token"}).
		Post("/api/token")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected 400 for an ownerless token, got %d %s",
			resp.StatusCode(), resp.String())
	}

	// Not logged in.
	resp, err = bearerReq("not-a-token").
		SetBody(api.PostSelfTokenInput{Description: "anonymous"}).
		Post("/api/token")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusUnauthorized {
		t.Fatalf("expected 401 for an unknown token, got %d %s",
			resp.StatusCode(), resp.String())
	}
}

func Test_28_5_Plain_user_self_token(t *testing.T) {
	c := loginPlain(t, selfUserUser, selfUserPass)

	var out api.PostTokenOutput
	resp, err := c.R().
		SetBody(api.PostSelfTokenInput{Description: "self-user's own token"}).
		SetResult(&out).
		Post("/api/token")
	if err != nil || resp.StatusCode() != http.StatusOK || out.Token == "" {
		t.Fatalf("self-user minting its own token failed: %v %s", err, resp)
	}
	test28UserToken = out.Token

	me := meOf(t, test28UserToken)
	if me.AccessTokenClaim.Resource.Id != test28UserId {
		t.Fatalf("expected token for user %d, got %d",
			test28UserId, me.AccessTokenClaim.Resource.Id)
	}
	if me.IsAdmin {
		t.Fatal("self token of a plain user must not be a global admin")
	}

	// The token grants nothing beyond the user's own login.
	resp, err = bearerReq(test28UserToken).
		SetHeader("X-Namespace", "default").
		Get("/api/admin/submissions")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("plain user's self token on an admin route: expected 403, "+
			"got %d", resp.StatusCode())
	}

	// A user token can mint another token for the same user, never for
	// anyone else.
	out = api.PostTokenOutput{}
	resp, err = bearerReq(test28UserToken).
		SetBody(api.PostSelfTokenInput{Description: "minted from a token"}).
		SetResult(&out).
		Post("/api/token")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("minting from a user token failed: %v %s", err, resp)
	}
	if got := meOf(t, out.Token).AccessTokenClaim.Resource.Id; got != test28UserId {
		t.Fatalf("token minted from a user token belongs to %d, want %d",
			got, test28UserId)
	}

	resp, err = bearerReq(test28UserToken).
		SetBody(api.PostTokenInput{
			TokenType:   "user",
			Description: "for someone else",
			UserId:      test28NsAdminId,
		}).
		Post("/api/admin/token")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("plain user minting a token for another user: expected 403, "+
			"got %d", resp.StatusCode())
	}

	// The token is listed for the global admin, bound to the user.
	var tokens []api.GetTokensOutput
	resp, err = adminReq().SetResult(&tokens).Get("/api/admin/tokens")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("get tokens failed: %v %s", err, resp)
	}
	found := false
	for _, tok := range tokens {
		if tok.Description == "self-user's own token" {
			found = true
			if tok.UserId != test28UserId {
				t.Fatalf("self token listed under user %d, want %d",
					tok.UserId, test28UserId)
			}
		}
	}
	if !found {
		t.Fatal("self token not listed in /api/admin/tokens")
	}
}

func Test_28_6_Namespace_admin_self_token(t *testing.T) {
	c := loginPlain(t, selfNsAdminUser, selfNsAdminPass)

	var out api.PostTokenOutput
	resp, err := c.R().
		SetBody(api.PostSelfTokenInput{Description: "self-nsadmin's own token"}).
		SetResult(&out).
		Post("/api/token")
	if err != nil || resp.StatusCode() != http.StatusOK || out.Token == "" {
		t.Fatalf("self-nsadmin minting its own token failed: %v %s", err, resp)
	}
	test28NsAdminToken = out.Token

	// Namespace admin of nsb through the token...
	resp, err = bearerReq(test28NsAdminToken).
		SetHeader("X-Namespace", nsB).
		Get("/api/admin/namespace/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("nsadmin self token on nsb admin route failed: %v %s", err, resp)
	}

	// ...but only a member of default.
	resp, err = bearerReq(test28NsAdminToken).
		SetHeader("X-Namespace", "default").
		Get("/api/admin/namespace/users")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("nsadmin self token on default admin route: expected 403, "+
			"got %d", resp.StatusCode())
	}

	// Naming a user, even itself, is the global-admin endpoint.
	for _, id := range []int64{test28NsAdminId, test28UserId} {
		resp, err = c.R().
			SetHeader("X-Namespace", nsB).
			SetBody(api.PostTokenInput{
				TokenType:   "user",
				Description: "explicit userId",
				UserId:      id,
			}).
			Post("/api/admin/token")
		if err != nil {
			t.Fatalf("request errored: %v", err)
		}
		if resp.StatusCode() != http.StatusForbidden {
			t.Fatalf("nsadmin minting a token for user %d via the admin "+
				"endpoint: expected 403, got %d", id, resp.StatusCode())
		}
	}
}

// Test_28_7_Self_token_follows_live_privileges: like admin-issued user
// tokens, a self token loses access as soon as the user does.
func Test_28_7_Self_token_follows_live_privileges(t *testing.T) {
	resp, err := adminReq().
		SetHeader("X-Namespace", nsB).
		SetBody(api.PutNamespaceUsersInput{
			UserIds: []int64{test28NsAdminId},
			IsAdmin: false,
		}).
		Put("/api/admin/namespace/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("demote self-nsadmin in nsb failed: %v %s", err, resp)
	}

	resp, err = bearerReq(test28NsAdminToken).
		SetHeader("X-Namespace", nsB).
		Get("/api/admin/namespace/users")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("self token after demotion: expected 403, got %d",
			resp.StatusCode())
	}

	// Restore for the CLI tests below.
	resp, err = adminReq().
		SetHeader("X-Namespace", nsB).
		SetBody(api.PutNamespaceUsersInput{
			UserIds: []int64{test28NsAdminId},
			IsAdmin: true,
		}).
		Put("/api/admin/namespace/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("re-promote self-nsadmin in nsb failed: %v %s", err, resp)
	}
}

func Test_28_8_CLI_user_login_with_username(t *testing.T) {
	home := t.TempDir()

	// Wrong password: fails, and saves no session.
	stdout, _, ok := runJudgeIn(t, home, "wrong-password\n",
		"user", "login",
		"--username", selfUserUser,
		"--endpoint", "http://localhost:8080")
	if ok || strings.Contains(stdout, "Login successful") {
		t.Fatalf("login with a wrong password should fail, got: %q", stdout)
	}
	sessionPath := filepath.Join(home, ".config", "judge", "session.json")
	if _, err := os.Stat(sessionPath); err == nil {
		t.Fatal("a failed login must not save a session")
	}

	// Right password: logs in (prompted once - stdin holds a single line)
	// and saves the session.
	stdout, stderr, ok := runJudgeIn(t, home, selfUserPass+"\n",
		"user", "login",
		"-u", selfUserUser,
		"--endpoint", "http://localhost:8080")
	if !ok || !strings.Contains(stdout, "Username: "+selfUserUser) {
		t.Fatalf("judge user login -u failed: %q\nstderr: %s", stdout, stderr)
	}

	b, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("session not saved: %v", err)
	}
	var session struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(b, &session); err != nil || session.RefreshToken == "" {
		t.Fatalf("saved session has no refresh token: %v %s", err, b)
	}

	// The saved session is a working login for that user - what later
	// `judge user` commands reuse without --username.
	c := resty.New().SetBaseURL("http://localhost:8080")
	c.SetCookie(&http.Cookie{Name: "refresh_token", Value: session.RefreshToken})
	var me api.MeOutput
	resp, err := c.R().SetResult(&me).Get("/api/me")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("saved session does not authenticate: %v %s", err, resp)
	}
	if me.AccessTokenClaim.Resource.Id != test28UserId {
		t.Fatalf("saved session belongs to %d, want %d",
			me.AccessTokenClaim.Resource.Id, test28UserId)
	}
}

func Test_28_9_CLI_admin_gen_own_token_with_username(t *testing.T) {
	home := t.TempDir()

	stdout, stderr, ok := runJudgeIn(t, home, selfNsAdminPass+"\n",
		"admin", "gen", "token", "user",
		"--username", selfNsAdminUser,
		"--description", "self-nsadmin via cli",
		"--endpoint", "http://localhost:8080")
	token, found := tokenFromOutput(stdout)
	if !ok || !found {
		t.Fatalf("judge admin -u gen token user failed: %q\nstderr: %s",
			stdout, stderr)
	}
	if got := meOf(t, token).AccessTokenClaim.Resource.Id; got != test28NsAdminId {
		t.Fatalf("CLI self token belongs to %d, want %d", got, test28NsAdminId)
	}

	// The minted token works with --access-token on namespace-admin commands.
	stdout, stderr, ok = runJudgeIn(t, home, "",
		"admin", "namespace", "users",
		"-n", nsB,
		"--access-token", token,
		"--endpoint", "http://localhost:8080")
	if !ok || !strings.Contains(stdout, selfNsAdminUser) {
		t.Fatalf("nsadmin CLI token listing nsb members failed: %q\nstderr: %s",
			stdout, stderr)
	}

	// A namespace admin cannot mint a token for another user.
	stdout, _, ok = runJudgeIn(t, home, selfNsAdminPass+"\n",
		"admin", "gen", "token", "user",
		strconv.FormatInt(test28UserId, 10),
		"-u", selfNsAdminUser,
		"--description", "for someone else",
		"--endpoint", "http://localhost:8080")
	if _, found := tokenFromOutput(stdout); ok || found {
		t.Fatalf("nsadmin minting a token for another user should fail, "+
			"got: %q", stdout)
	}
}
