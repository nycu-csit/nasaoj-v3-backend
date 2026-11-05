package tests

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"go.yaml.in/yaml/v3"
)

// A namespace admin can do, within the namespaces it administers, what a
// global admin can: manage problems/tools, submit as members, and see and
// manage every submission. nsx-admin administers nsx; nsy is out of reach.

const (
	test27Nsx = "nsx"
	test27Nsy = "nsy"
)

var (
	test27AdminId  int64 // admin of nsx
	test27MemberId int64 // plain member of nsx
	test27OtherId  int64 // plain member of nsy

	test27AdminToken  string
	test27MemberToken string

	test27NsxProblemSHA1 string
	test27NsyProblemSHA1 string
	test27NsySubId       int64
)

// test27Req returns a request as nsx-admin targeting the given namespace.
func test27Req(namespace string) *resty.Request {
	return bearerReq(test27AdminToken).SetHeader("X-Namespace", namespace)
}

// test27Judge runs the admin CLI as nsx-admin in nsx.
func test27Judge(t *testing.T, args ...string) string {
	t.Helper()
	return runJudge(t, append(args,
		"-n", test27Nsx,
		"--access-token", test27AdminToken,
		"--endpoint", "http://localhost:8080")...)
}

// test27FirstSHA1 returns the sha1 column of an import's first output line.
func test27FirstSHA1(t *testing.T, out string) string {
	t.Helper()
	fields := strings.Fields(out)
	if len(fields) == 0 {
		t.Fatalf("no sha1 in import output: %q", out)
	}
	return fields[0]
}

func test27ExpectStatus(t *testing.T, what string, resp *resty.Response, err error, want int) {
	t.Helper()
	if err != nil || resp.StatusCode() != want {
		t.Fatalf("%s: expected %d, got %v %v", what, want, err, resp)
	}
}

func test27Problems(t *testing.T, req *resty.Request) []api.ProblemOutput {
	t.Helper()
	var problems []api.ProblemOutput
	resp, err := req.SetResult(&problems).Get("/api/admin/problems")
	test27ExpectStatus(t, "get problems", resp, err, http.StatusOK)
	return problems
}

func test27Tools(t *testing.T, req *resty.Request) []api.ToolOutput {
	t.Helper()
	var tools []api.ToolOutput
	resp, err := req.SetResult(&tools).Get("/api/admin/tools")
	test27ExpectStatus(t, "get tools", resp, err, http.StatusOK)
	return tools
}

func test27Submit(t *testing.T, req *resty.Request, sha1 string, ownerId int64) *resty.Response {
	t.Helper()
	resp, err := req.
		SetBody(api.AdminSubmitInput{
			AsOwnerId:   ownerId,
			SubmitInput: api.SubmitInput{RefSHA1: sha1, Type: models.RefTypeProblem},
		}).
		Post("/api/admin/submit")
	if err != nil {
		t.Fatalf("submit errored: %v", err)
	}
	return resp
}

func test27SubId(t *testing.T, resp *resty.Response) int64 {
	t.Helper()
	test27ExpectStatus(t, "submit", resp, nil, http.StatusOK)
	var out api.SubmitOutput
	if err := json.Unmarshal(resp.Body(), &out); err != nil {
		t.Fatalf("decode submit output: %v", err)
	}
	return out.Id
}

func test27WaitDone(t *testing.T, req func() *resty.Request, subId int64) api.ResultOutput {
	t.Helper()
	for range 60 {
		var result api.ResultOutput
		resp, err := req().
			SetResult(&result).
			SetQueryParam("subId", strconv.FormatInt(subId, 10)).
			Get("/api/admin/results")
		test27ExpectStatus(t, "get results", resp, err, http.StatusOK)
		if len(result.Jobs) > 0 && !slices.ContainsFunc(result.Jobs,
			func(j api.ResultOutputJob) bool {
				switch j.Status {
				case models.StatusPending, models.StatusBlocked, models.StatusRunning:
					return true
				}
				return false
			}) {
			return result
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("submission %d did not finish", subId)
	return api.ResultOutput{}
}

func Test_27_1_Setup_namespaces_and_users(t *testing.T) {
	for _, ns := range []string{test27Nsx, test27Nsy} {
		resp, err := adminReq().
			SetBody(map[string]any{"name": ns, "title": "Namespace " + ns}).
			Post("/api/admin/namespace")
		test27ExpectStatus(t, "create "+ns, resp, err, http.StatusOK)
	}

	runJudgeStdin(t, `---
- username: nsx-admin
  password: nsx-admin-pass
- username: nsx-member
  password: nsx-member-pass
- username: nsy-member
  password: nsy-member-pass
`, "admin", "import", "users", "plain",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var users []struct {
		Id       int64  `json:"id"`
		Username string `json:"username"`
	}
	resp, err := adminReq().SetResult(&users).Get("/api/admin/users")
	test27ExpectStatus(t, "get users", resp, err, http.StatusOK)
	for _, u := range users {
		switch u.Username {
		case "nsx-admin":
			test27AdminId = u.Id
		case "nsx-member":
			test27MemberId = u.Id
		case "nsy-member":
			test27OtherId = u.Id
		}
	}
	if test27AdminId == 0 || test27MemberId == 0 || test27OtherId == 0 {
		t.Fatalf("users not found: %+v", users)
	}

	for _, m := range []struct {
		ns      string
		id      int64
		isAdmin bool
	}{
		{test27Nsx, test27AdminId, true},
		{test27Nsx, test27MemberId, false},
		{test27Nsy, test27OtherId, false},
	} {
		resp, err := adminReq().
			SetHeader("X-Namespace", m.ns).
			SetBody(map[string]any{"userIds": []int64{m.id}, "isAdmin": m.isAdmin}).
			Put("/api/admin/namespace/users")
		test27ExpectStatus(t, "add member to "+m.ns, resp, err, http.StatusOK)
	}

	test27AdminToken = genToken(t, "nsx-admin's token",
		"user", strconv.FormatInt(test27AdminId, 10))
	test27MemberToken = genToken(t, "nsx-member's token",
		"user", strconv.FormatInt(test27MemberId, 10))

	// nsy's own content, set up by the global admin: a problem with a
	// submission, and a tool sharing its toolId with the one nsx imports.
	test27NsyProblemSHA1 = test27FirstSHA1(t, runJudge(t,
		"admin", "import", "problems",
		"--problems-dir", "resources/27/nsy-problems",
		"--testcases-dir", "resources/27/testcases",
		"-n", test27Nsy,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080"))
	nsyToolSHA1 := test27FirstSHA1(t, runJudge(t,
		"admin", "import", "tools",
		"--tools-dir", "resources/27/nsy-tools",
		"--testcases-dir", "resources/27/testcases",
		"-n", test27Nsy,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080"))
	for _, args := range [][]string{
		{"problem", test27NsyProblemSHA1},
		{"tool", nsyToolSHA1},
	} {
		runJudge(t, append([]string{"admin", "enable"}, append(args,
			"-n", test27Nsy,
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")...)...)
	}
	test27NsySubId = test27SubId(t, test27Submit(t,
		adminReq().SetHeader("X-Namespace", test27Nsy),
		test27NsyProblemSHA1, test27OtherId))
}

func Test_27_2_Import_and_enable_problem(t *testing.T) {
	test27NsxProblemSHA1 = test27FirstSHA1(t, test27Judge(t,
		"admin", "import", "problems",
		"--problems-dir", "resources/27/problems",
		"--testcases-dir", "resources/27/testcases"))
	test27Judge(t, "admin", "enable", "problem", test27NsxProblemSHA1)

	problems := test27Problems(t, test27Req(test27Nsx))
	if !slices.ContainsFunc(problems, func(p api.ProblemOutput) bool {
		return p.SHA1 == test27NsxProblemSHA1 && p.IsEnabled
	}) {
		t.Fatalf("expected enabled problem %s in nsx, got %+v",
			test27NsxProblemSHA1, problems)
	}
}

func Test_27_2_1_Same_problem_in_another_namespace(t *testing.T) {
	// Identical content imported into nsy is its own problem there, with a
	// different sha1 - not a reference to nsx's.
	sha1 := test27FirstSHA1(t, runJudge(t,
		"admin", "import", "problems",
		"--problems-dir", "resources/27/problems",
		"--testcases-dir", "resources/27/testcases",
		"-n", test27Nsy,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080"))
	if sha1 == test27NsxProblemSHA1 {
		t.Fatalf("same content got the same sha1 %s in nsx and nsy", sha1)
	}

	resp, err := adminReq().
		SetHeader("X-Namespace", test27Nsy).
		SetBody(api.PutProblemStateInput{ProblemSHA1: sha1, IsEnabled: true}).
		Put("/api/admin/problem/state")
	test27ExpectStatus(t, "enable nsy copy", resp, err, http.StatusOK)

	// Enabling nsy's copy leaves nsx's version enabled.
	if !slices.ContainsFunc(test27Problems(t, test27Req(test27Nsx)),
		func(p api.ProblemOutput) bool {
			return p.SHA1 == test27NsxProblemSHA1 && p.IsEnabled
		}) {
		t.Fatal("enabling nsy's copy disabled nsx's problem")
	}
}

func Test_27_3_Submit_as_member_and_read_everything(t *testing.T) {
	subId := test27SubId(t, test27Submit(t,
		test27Req(test27Nsx), test27NsxProblemSHA1, test27MemberId))
	test27WaitDone(t, func() *resty.Request { return test27Req(test27Nsx) }, subId)

	job := func(req *resty.Request) *resty.Request {
		return req.
			SetQueryParam("subId", strconv.FormatInt(subId, 10)).
			SetQueryParam("stage", "main").
			SetQueryParam("testcase", "work")
	}

	for path, want := range map[string]string{
		"/api/admin/stdout": "NSX_STDOUT",
		"/api/admin/stderr": "NSX_STDERR",
	} {
		resp, err := job(test27Req(test27Nsx)).Get(path)
		test27ExpectStatus(t, path, resp, err, http.StatusOK)
		if !strings.Contains(resp.String(), want) {
			t.Fatalf("%s: expected %q, got %q", path, want, resp.String())
		}
	}

	var artifacts api.GetArtifactsURLOutput
	resp, err := job(test27Req(test27Nsx)).
		SetResult(&artifacts).
		Get("/api/admin/artifacts/url")
	test27ExpectStatus(t, "artifacts url", resp, err, http.StatusOK)
	if artifacts.URL == "" {
		t.Fatal("expected a presigned artifacts url")
	}

	// The plain member who owns the submission still can't read stderr.
	resp, err = job(bearerReq(test27MemberToken).
		SetHeader("X-Namespace", test27Nsx)).
		Get("/api/admin/stderr")
	test27ExpectStatus(t, "member reading stderr", resp, err, http.StatusForbidden)
}

func Test_27_4_Submit_as_non_member_refused(t *testing.T) {
	resp := test27Submit(t,
		test27Req(test27Nsx), test27NsxProblemSHA1, test27OtherId)
	test27ExpectStatus(t, "submit as nsy member", resp, nil, http.StatusForbidden)
}

func Test_27_5_Cancel_and_delete_any_submission(t *testing.T) {
	subId := test27SubId(t, test27Submit(t,
		test27Req(test27Nsx), test27NsxProblemSHA1, test27MemberId))
	sub := strconv.FormatInt(subId, 10)

	resp, err := test27Req(test27Nsx).
		SetBody(api.PutCancelSubmissionInput{SubId: subId}).
		Put("/api/admin/submission/cancel")
	test27ExpectStatus(t, "cancel", resp, err, http.StatusOK)
	test27WaitDone(t, func() *resty.Request { return test27Req(test27Nsx) }, subId)

	resp, err = test27Req(test27Nsx).
		SetQueryParam("subId", sub).
		Delete("/api/admin/submission")
	test27ExpectStatus(t, "delete", resp, err, http.StatusOK)

	resp, err = test27Req(test27Nsx).
		SetQueryParam("subId", sub).
		Get("/api/admin/results")
	test27ExpectStatus(t, "results after delete", resp, err, http.StatusNotFound)
}

func Test_27_6_Other_namespace_out_of_reach(t *testing.T) {
	// Naming nsy outright is refused.
	resp, err := test27Req(test27Nsy).Get("/api/admin/problems")
	test27ExpectStatus(t, "nsy problems", resp, err, http.StatusForbidden)

	// Reaching nsy's submission through nsx is refused too.
	sub := strconv.FormatInt(test27NsySubId, 10)
	resp, err = test27Req(test27Nsx).
		SetQueryParam("subId", sub).
		Get("/api/admin/results")
	test27ExpectStatus(t, "nsy results", resp, err, http.StatusNotFound)
	resp, err = test27Req(test27Nsx).
		SetBody(api.PutCancelSubmissionInput{SubId: test27NsySubId}).
		Put("/api/admin/submission/cancel")
	test27ExpectStatus(t, "cancel nsy submission", resp, err, http.StatusNotFound)
	resp, err = test27Req(test27Nsx).
		SetQueryParam("subId", sub).
		Delete("/api/admin/submission")
	test27ExpectStatus(t, "delete nsy submission", resp, err, http.StatusNotFound)

	// nsy's problem can't be enabled, disabled or deleted through nsx - its
	// sha1 is only looked up among nsx's problems.
	resp, err = test27Req(test27Nsx).
		SetBody(api.PutProblemStateInput{
			ProblemSHA1: test27NsyProblemSHA1, IsEnabled: true,
		}).
		Put("/api/admin/problem/state")
	test27ExpectStatus(t, "enable nsy problem", resp, err, http.StatusNotFound)
	test27Req(test27Nsx).
		SetBody(api.PutProblemStateInput{
			ProblemSHA1: test27NsyProblemSHA1, IsEnabled: false,
		}).
		Put("/api/admin/problem/state")
	test27Req(test27Nsx).
		SetQueryParam("problemSHA1", test27NsyProblemSHA1).
		Delete("/api/admin/problem")

	nsy := adminReq().SetHeader("X-Namespace", test27Nsy)
	if !slices.ContainsFunc(test27Problems(t, nsy), func(p api.ProblemOutput) bool {
		return p.SHA1 == test27NsyProblemSHA1 && p.IsEnabled
	}) {
		t.Fatal("nsy's problem was disabled or deleted through nsx")
	}
	resp, err = adminReq().
		SetHeader("X-Namespace", test27Nsy).
		SetQueryParam("subId", sub).
		Get("/api/admin/results")
	test27ExpectStatus(t, "nsy submission still there", resp, err, http.StatusOK)
}

func Test_27_7_Manage_tool_without_touching_other_namespace(t *testing.T) {
	sha1 := test27FirstSHA1(t, test27Judge(t,
		"admin", "import", "tools",
		"--tools-dir", "resources/27/tools",
		"--testcases-dir", "resources/27/testcases"))
	test27Judge(t, "admin", "enable", "tool", sha1)

	if !slices.ContainsFunc(test27Tools(t, test27Req(test27Nsx)),
		func(tool api.ToolOutput) bool { return tool.SHA1 == sha1 && tool.IsEnabled }) {
		t.Fatalf("expected tool %s enabled in nsx", sha1)
	}

	// Enabling nsx's tool only disables other versions within nsx, not
	// nsy's tool that happens to share the toolId.
	nsy := func() *resty.Request {
		return adminReq().SetHeader("X-Namespace", test27Nsy)
	}
	if !slices.ContainsFunc(test27Tools(t, nsy()),
		func(tool api.ToolOutput) bool { return tool.ToolId == 2701 && tool.IsEnabled }) {
		t.Fatal("enabling nsx's tool disabled nsy's tool with the same toolId")
	}

	resp, err := test27Req(test27Nsx).
		SetBody(api.PutToolStateInput{ToolSHA1: sha1, IsEnabled: false}).
		Put("/api/admin/tool/state")
	test27ExpectStatus(t, "disable tool", resp, err, http.StatusOK)
	resp, err = test27Req(test27Nsx).
		SetQueryParam("toolSHA1", sha1).
		Delete("/api/admin/tool")
	test27ExpectStatus(t, "delete tool", resp, err, http.StatusOK)

	if slices.ContainsFunc(test27Tools(t, test27Req(test27Nsx)),
		func(tool api.ToolOutput) bool { return tool.SHA1 == sha1 }) {
		t.Fatalf("expected tool %s deleted", sha1)
	}
}

func Test_27_8_Disable_and_delete_problem(t *testing.T) {
	resp, err := test27Req(test27Nsx).
		SetBody(api.PutProblemStateInput{
			ProblemSHA1: test27NsxProblemSHA1, IsEnabled: false,
		}).
		Put("/api/admin/problem/state")
	test27ExpectStatus(t, "disable problem", resp, err, http.StatusOK)
	if slices.ContainsFunc(test27Problems(t, test27Req(test27Nsx)),
		func(p api.ProblemOutput) bool {
			return p.SHA1 == test27NsxProblemSHA1 && p.IsEnabled
		}) {
		t.Fatal("expected problem disabled")
	}

	resp, err = test27Req(test27Nsx).
		SetQueryParam("problemSHA1", test27NsxProblemSHA1).
		Delete("/api/admin/problem")
	test27ExpectStatus(t, "delete problem", resp, err, http.StatusOK)
	if slices.ContainsFunc(test27Problems(t, test27Req(test27Nsx)),
		func(p api.ProblemOutput) bool { return p.SHA1 == test27NsxProblemSHA1 }) {
		t.Fatal("expected problem deleted")
	}
}

func Test_27_9_Global_only_routes_refused(t *testing.T) {
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/api/admin/namespace", map[string]any{"name": "nsz", "title": "Z"}},
		{http.MethodPut, "/api/admin/namespace/state", map[string]any{"id": 1, "isEnabled": false}},
		{http.MethodGet, "/api/admin/users", nil},
		{http.MethodGet, "/api/admin/tokens", nil},
		{http.MethodGet, "/api/admin/worker/states", nil},
		{http.MethodPost, "/api/admin/token", map[string]any{"description": "x"}},
	} {
		req := test27Req(test27Nsx)
		if c.body != nil {
			req.SetBody(c.body)
		}
		resp, err := req.Execute(c.method, c.path)
		test27ExpectStatus(t, c.method+" "+c.path, resp, err, http.StatusForbidden)
	}
}

func Test_27_10_Content_file_sha1_must_match_payload(t *testing.T) {
	payload := "echo test27 content\n"
	h := sha1.Sum([]byte(payload))
	good := hex.EncodeToString(h[:])

	resp, err := test27Req(test27Nsx).
		SetBody([]api.PutContentFilesInput{
			{SHA1: strings.Repeat("0", 40), Name: "x.sh", Payload: payload},
		}).
		Put("/api/admin/content-files")
	test27ExpectStatus(t, "mismatched sha1", resp, err, http.StatusBadRequest)

	resp, err = test27Req(test27Nsx).
		SetBody([]api.PutContentFilesInput{
			{SHA1: good, Name: "x.sh", Payload: payload},
		}).
		Put("/api/admin/content-files")
	test27ExpectStatus(t, "matching sha1", resp, err, http.StatusOK)
}

// test27Me returns /api/me's namespaces for a plain-login user, keyed by
// namespace name with the namespace admin flag as value.
func test27Me(t *testing.T, username, password string) (bool, map[string]bool) {
	t.Helper()
	var me api.MeOutput
	resp, err := loginPlain(t, username, password).R().
		SetResult(&me).
		Get("/api/me")
	test27ExpectStatus(t, username+" /api/me", resp, err, http.StatusOK)
	namespaces := make(map[string]bool)
	for _, ns := range me.Namespaces {
		namespaces[ns.Name] = ns.IsAdmin
	}
	return me.IsAdmin, namespaces
}

func Test_27_11_Import_users_into_namespace(t *testing.T) {
	out := runJudgeStdin(t, `---
- username: nsx-new
  password: nsx-new-pass
- username: nsx-new-admin
  password: nsx-new-admin-pass
  isAdmin: true
- username: nsy-member
  password: hijacked-pass
`, "admin", "namespace", "import-users", "plain",
		"-n", test27Nsx,
		"--access-token", test27AdminToken,
		"--endpoint", "http://localhost:8080")

	var imported []models.NamespaceUserImported
	if err := yaml.Unmarshal([]byte(out), &imported); err != nil {
		t.Fatalf("decode import output %q: %v", out, err)
	}
	created := make(map[string]bool)
	for _, u := range imported {
		created[u.Username] = u.Created
	}
	if !created["nsx-new"] || !created["nsx-new-admin"] || created["nsy-member"] {
		t.Fatalf("expected the two new users created and nsy-member not, got %+v",
			imported)
	}

	// New users join nsx plus the default namespace every user starts in,
	// and the isAdmin flag is namespace admin of nsx, never global admin.
	for _, u := range []struct {
		username, password string
		nsxAdmin           bool
	}{
		{"nsx-new", "nsx-new-pass", false},
		{"nsx-new-admin", "nsx-new-admin-pass", true},
	} {
		globalAdmin, namespaces := test27Me(t, u.username, u.password)
		nsxAdmin, inNsx := namespaces[test27Nsx]
		defaultAdmin, inDefault := namespaces["default"]
		if globalAdmin || len(namespaces) != 2 || !inNsx || !inDefault ||
			nsxAdmin != u.nsxAdmin || defaultAdmin {
			t.Fatalf("%s: expected nsx (admin=%v) and default, got admin=%v %v",
				u.username, u.nsxAdmin, globalAdmin, namespaces)
		}
	}

	// The existing user was only added to nsx: its password is unchanged.
	resp, err := resty.New().SetBaseURL("http://localhost:8080").R().
		SetBody(map[string]string{"username": "nsy-member", "password": "hijacked-pass"}).
		Post("/auth/login/plain")
	if err != nil || resp.StatusCode() == http.StatusOK {
		t.Fatalf("import changed nsy-member's password: %v %v", err, resp)
	}
	_, namespaces := test27Me(t, "nsy-member", "nsy-member-pass")
	if _, ok := namespaces[test27Nsx]; !ok {
		t.Fatalf("expected nsy-member added to nsx, got %v", namespaces)
	}
	if _, ok := namespaces[test27Nsy]; !ok {
		t.Fatalf("expected nsy-member to stay in nsy, got %v", namespaces)
	}
}

func Test_27_12_Import_CSSSO_users_into_namespace(t *testing.T) {
	var imported []models.NamespaceUserImported
	resp, err := test27Req(test27Nsx).
		SetBody([]api.PutNamespaceUsersCSSSOInputUser{{CsId: "nsx-cs"}}).
		SetResult(&imported).
		Put("/api/admin/namespace/users/cssso")
	test27ExpectStatus(t, "import cssso", resp, err, http.StatusOK)
	if len(imported) != 1 || !imported[0].Created {
		t.Fatalf("expected nsx-cs created, got %+v", imported)
	}

	var members []models.NamespaceMember
	resp, err = test27Req(test27Nsx).
		SetResult(&members).
		Get("/api/admin/namespace/users")
	test27ExpectStatus(t, "list members", resp, err, http.StatusOK)
	if !slices.ContainsFunc(members, func(m models.NamespaceMember) bool {
		return m.UserId == imported[0].Id && m.Username == "nsx-cs"
	}) {
		t.Fatalf("expected nsx-cs (%d) in nsx, got %+v", imported[0].Id, members)
	}
}

func Test_27_13_Import_users_refused_outside_own_namespaces(t *testing.T) {
	body := []api.PutNamespaceUsersPlainInputUser{
		{Username: "nsx-sneaky", Password: "sneaky-pass"},
	}

	resp, err := test27Req(test27Nsy).
		SetBody(body).
		Put("/api/admin/namespace/users/plain")
	test27ExpectStatus(t, "import into nsy", resp, err, http.StatusForbidden)

	resp, err = bearerReq(test27MemberToken).
		SetHeader("X-Namespace", test27Nsx).
		SetBody(body).
		Put("/api/admin/namespace/users/plain")
	test27ExpectStatus(t, "member importing", resp, err, http.StatusForbidden)
}
