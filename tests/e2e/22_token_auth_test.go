package tests

import (
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
)

// This suite covers static bearer-token authorization: a token is either a
// global-admin/worker token (privileges baked in at issuance) or a "user"
// token, whose privileges and namespace membership are resolved live from
// the owning user on every request instead of being frozen into the token.
//
// It reuses the "nsb" namespace and problem 2101 set up by 21_namespace_test.go.

const (
	tokNsAdminUser = "tok-nsadmin"
	tokNsAdminPass = "tok-nsadmin-pass"
	tokUserUser    = "tok-user"
	tokUserPass    = "tok-user-pass"
)

var (
	test22NsAdminId int64
	test22UserId    int64

	test22NsAdminToken string
	test22UserToken    string
)

// genToken creates a token via the CLI and returns the raw token string.
func genToken(t *testing.T, description string, args ...string) string {
	t.Helper()

	fullArgs := append([]string{
		"admin", "gen", "token",
	}, args...)
	fullArgs = append(fullArgs,
		"--description", description,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	out := runJudge(t, fullArgs...)

	_, rest, ok := strings.Cut(out, "Token: ")
	if !ok {
		t.Fatalf("no token in output: %q", out)
	}
	return strings.Fields(rest)[0]
}

// bearerReq returns a request authenticated with the given raw bearer token.
func bearerReq(token string) *resty.Request {
	return client.R().SetHeader("Authorization", "Bearer "+token)
}

func Test_22_1_Create_token_test_users(t *testing.T) {
	stdin := `---
- username: '` + tokNsAdminUser + `'
  password: '` + tokNsAdminPass + `'
- username: '` + tokUserUser + `'
  password: '` + tokUserPass + `'
`
	runJudgeStdin(t, stdin,
		"admin", "import", "users", "plain",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var users []struct {
		Id       int64  `json:"id"`
		Username string `json:"username"`
	}
	resp, err := adminReq().SetResult(&users).Get("/api/admin/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("get users failed: %v %s", err, resp)
	}
	for _, u := range users {
		switch u.Username {
		case tokNsAdminUser:
			test22NsAdminId = u.Id
		case tokUserUser:
			test22UserId = u.Id
		}
	}
	if test22NsAdminId == 0 || test22UserId == 0 {
		t.Fatalf("token test users not found (nsadmin=%d user=%d)",
			test22NsAdminId, test22UserId)
	}
}

func Test_22_2_Add_to_nsb(t *testing.T) {
	resp, err := adminReq().
		SetHeader("X-Namespace", nsB).
		SetBody(map[string]interface{}{
			"userIds": []int64{test22NsAdminId},
			"isAdmin": true,
		}).
		Put("/api/admin/namespace/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("add tok-nsadmin to nsb failed: %v %s", err, resp)
	}

	resp, err = adminReq().
		SetHeader("X-Namespace", nsB).
		SetBody(map[string]interface{}{
			"userIds": []int64{test22UserId},
			"isAdmin": false,
		}).
		Put("/api/admin/namespace/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("add tok-user to nsb failed: %v %s", err, resp)
	}

	// Confine both to nsb: every user starts in default.
	removeFromDefault(t, test22NsAdminId, test22UserId)
}

func Test_22_3_Gen_user_token_validation(t *testing.T) {
	// Missing description.
	resp, err := adminReq().
		SetBody(api.PostTokenInput{
			TokenType: "user", UserId: test22UserId,
		}).
		Post("/api/admin/token")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing description, got %d %s",
			resp.StatusCode(), resp.String())
	}

	// Missing userId.
	resp, err = adminReq().
		SetBody(api.PostTokenInput{
			TokenType: "user", Description: "missing userId",
		}).
		Post("/api/admin/token")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing userId, got %d %s",
			resp.StatusCode(), resp.String())
	}

	// Unknown userId.
	resp, err = adminReq().
		SetBody(api.PostTokenInput{
			TokenType: "user", Description: "unknown user", UserId: 999999999,
		}).
		Post("/api/admin/token")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown userId, got %d %s",
			resp.StatusCode(), resp.String())
	}

	// A non-admin cannot mint tokens for anyone, including themselves.
	c := loginPlain(t, tokUserUser, tokUserPass)
	resp, err = c.R().
		SetBody(api.PostTokenInput{
			TokenType: "user", Description: "self-service", UserId: test22UserId,
		}).
		Post("/api/admin/token")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin minting a token, got %d",
			resp.StatusCode())
	}
}

func Test_22_4_Plain_member_user_token_scoped_to_nsb(t *testing.T) {
	test22UserToken = genToken(t, "tok-user's live token",
		"user", strconv.FormatInt(test22UserId, 10))

	// Member of nsb: can read nsb problems.
	var nsbProblems []api.ProblemOutput
	resp, err := bearerReq(test22UserToken).
		SetHeader("X-Namespace", nsB).
		SetResult(&nsbProblems).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("tok-user token get problems (nsb) failed: %v %s", err, resp)
	}
	if !containsProblemId(nsbProblems, 2101) {
		t.Fatalf("expected problem 2101 via tok-user token, got %+v", nsbProblems)
	}

	// Not a member of default: forbidden.
	resp, err = bearerReq(test22UserToken).
		SetHeader("X-Namespace", "default").
		Get("/api/problems")
	if err != nil {
		t.Fatalf("tok-user token get problems (default) errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("tok-user token reading default: expected 403, got %d %s",
			resp.StatusCode(), resp.String())
	}

	// Plain member cannot manage namespace membership.
	resp, err = bearerReq(test22UserToken).
		SetHeader("X-Namespace", nsB).
		SetBody(map[string]interface{}{
			"userIds": []int64{test22UserId},
			"isAdmin": true,
		}).
		Put("/api/admin/namespace/users")
	if err != nil {
		t.Fatalf("tok-user token manage members errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("tok-user token managing members: expected 403, got %d",
			resp.StatusCode())
	}
}

func Test_22_5_Namespace_admin_user_token(t *testing.T) {
	test22NsAdminToken = genToken(t, "tok-nsadmin's live token",
		"user", strconv.FormatInt(test22NsAdminId, 10))

	// Admin of nsb: can manage nsb membership.
	resp, err := bearerReq(test22NsAdminToken).
		SetHeader("X-Namespace", nsB).
		SetBody(map[string]interface{}{
			"userIds": []int64{test22UserId},
			"isAdmin": false,
		}).
		Put("/api/admin/namespace/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("tok-nsadmin token manage members (nsb) failed: %v %s", err, resp)
	}

	// Not admin of default: forbidden there.
	resp, err = bearerReq(test22NsAdminToken).
		SetHeader("X-Namespace", "default").
		SetBody(map[string]interface{}{
			"userIds": []int64{test22UserId},
			"isAdmin": false,
		}).
		Put("/api/admin/namespace/users")
	if err != nil {
		t.Fatalf("tok-nsadmin token manage members (default) errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("tok-nsadmin token managing default members: expected 403, got %d",
			resp.StatusCode())
	}

	// Namespace admin is not a global admin: cannot create namespaces.
	resp, err = bearerReq(test22NsAdminToken).
		SetBody(map[string]interface{}{"name": "tokns", "title": "Token NS"}).
		Post("/api/admin/namespace")
	if err != nil {
		t.Fatalf("tok-nsadmin token create namespace errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("tok-nsadmin token creating namespace: expected 403, got %d",
			resp.StatusCode())
	}
}

// Test_22_6_Live_privileges_without_reissue is the key behavioral difference
// from JWT sessions: a user token is never reissued here, yet membership
// changes made after it was minted take effect on the very next request.
func Test_22_6_Live_privileges_without_reissue(t *testing.T) {
	// tok-user's token was minted before it had any access to "default".
	resp, err := bearerReq(test22UserToken).
		SetHeader("X-Namespace", "default").
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("expected tok-user token still forbidden from default, got %v %d",
			err, resp.StatusCode())
	}

	// Admin grants membership in "default" - no token reissue happens.
	resp, err = adminReq().
		SetHeader("X-Namespace", "default").
		SetBody(map[string]interface{}{
			"userIds": []int64{test22UserId},
			"isAdmin": false,
		}).
		Put("/api/admin/namespace/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("add tok-user to default failed: %v %s", err, resp)
	}

	// The *same* already-issued token now sees "default" immediately.
	var defaultProblems []api.ProblemOutput
	resp, err = bearerReq(test22UserToken).
		SetHeader("X-Namespace", "default").
		SetResult(&defaultProblems).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected tok-user token to see default after grant "+
			"without reissue: %v (status %d) %s", err, resp.StatusCode(), resp)
	}
	if !containsProblemId(defaultProblems, 2102) {
		t.Fatalf("expected problem 2102 via tok-user token, got %+v", defaultProblems)
	}

	// Admin revokes membership - again no token reissue.
	resp, err = adminReq().
		SetHeader("X-Namespace", "default").
		SetBody(map[string]interface{}{
			"userIds": []int64{test22UserId},
		}).
		Delete("/api/admin/namespace/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("remove tok-user from default failed: %v %s", err, resp)
	}

	// The same token immediately loses access again.
	resp, err = bearerReq(test22UserToken).
		SetHeader("X-Namespace", "default").
		Get("/api/problems")
	if err != nil {
		t.Fatalf("tok-user token get problems (default) errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("expected revoked tok-user token to be forbidden "+
			"immediately, got %d %s", resp.StatusCode(), resp.String())
	}
}

func Test_22_7_Disable_enable_user_token(t *testing.T) {
	resp, err := bearerReq(test22UserToken).Get("/api/me")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("token should work before disable: %v %s", err, resp)
	}

	runJudge(t,
		"admin", "disable", "token", test22UserToken,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	resp, err = bearerReq(test22UserToken).Get("/api/me")
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if resp.StatusCode() == http.StatusOK {
		t.Fatal("expected disabled user token to be rejected")
	}

	runJudge(t,
		"admin", "enable", "token", test22UserToken,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	resp, err = bearerReq(test22UserToken).Get("/api/me")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("re-enabled user token should work: %v %s", err, resp)
	}
}

func Test_22_8_Token_listing_shows_userId_and_description(t *testing.T) {
	var tokens []api.GetTokensOutput
	resp, err := adminReq().SetResult(&tokens).Get("/api/admin/tokens")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("get tokens failed: %v %s", err, resp)
	}

	found := false
	for _, tok := range tokens {
		if tok.UserId == test22UserId {
			found = true
			if tok.Description != "tok-user's live token" {
				t.Fatalf("expected description to survive round-trip, got %q",
					tok.Description)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected a listed token bound to user %d", test22UserId)
	}
}

func Test_22_9_Gen_token_requires_description(t *testing.T) {
	// judge.go ignores RootCmd.Execute()'s error and always exits 0, so a
	// missing required flag doesn't fail the process - assert on cobra's
	// error output and the absence of an issued token instead.
	cmd := exec.Command(
		"judge", "admin", "gen", "token", "worker",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("judge gen token worker errored: %v\nstderr: %s", err, stderr.String())
	}

	if !strings.Contains(stderr.String(), `required flag(s) "description" not set`) {
		t.Fatalf("expected a required-flag error, got stderr: %q", stderr.String())
	}
	if strings.Contains(stdout.String(), "Token: ") {
		t.Fatalf("expected no token to be issued without --description, got: %q",
			stdout.String())
	}
}
