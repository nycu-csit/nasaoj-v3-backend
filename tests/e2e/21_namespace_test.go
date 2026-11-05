package tests

import (
	"net/http"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

const (
	nsB = "nsb"

	nsAdminUser = "nsb-admin"
	nsAdminPass = "nsb-admin-pass"
	nsUserUser  = "nsb-user"
	nsUserPass  = "nsb-user-pass"
)

var (
	test21NsAdminId int64
	test21NsUserId  int64
)

var test21NsbProblemSHA1 string

func Test_21_1_Create_namespace(t *testing.T) {
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+viper.GetString("admin-token")).
		SetBody(map[string]interface{}{
			"name":  nsB,
			"title": "Namespace B",
		}).
		Post("/api/admin/namespace")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("create namespace failed: %v (status %d) %s",
			err, resp.StatusCode(), resp.String())
	}
}

func Test_21_2_List_namespaces(t *testing.T) {
	var namespaces []models.Namespace
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+viper.GetString("admin-token")).
		SetResult(&namespaces).
		Get("/api/admin/namespaces")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("list namespaces failed: %v %s", err, resp)
	}

	var foundDefault, foundNsb bool
	for _, ns := range namespaces {
		if ns.Name == "default" {
			foundDefault = true
		}
		if ns.Name == nsB {
			foundNsb = true
		}
	}
	if !foundDefault || !foundNsb {
		t.Fatalf("expected default and %s namespaces, got %+v", nsB, namespaces)
	}
}

func Test_21_3_Import_problem_into_nsb(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "import", "problems",
		"--problems-dir", "resources/21/problems",
		"--testcases-dir", "resources/21/testcases",
		"--namespace", nsB,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("import into %s failed: %v\nstderr: %s", nsB, err, stderr.String())
	}

	for line := range strings.Lines(stdout.String()) {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			test21NsbProblemSHA1 = fields[0]
			break
		}
	}
	if test21NsbProblemSHA1 == "" {
		t.Fatalf("no sha1 returned from import: %q", stdout.String())
	}

	// Enable it.
	enable := exec.Command(
		"judge", "admin", "enable", "problem", test21NsbProblemSHA1,
		"--namespace", nsB,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")
	var estderr strings.Builder
	enable.Stderr = &estderr
	if err := enable.Run(); err != nil {
		t.Fatalf("enable in %s failed: %v\nstderr: %s", nsB, err, estderr.String())
	}
}

func Test_21_4_Problem_visible_in_nsb_only(t *testing.T) {
	// In nsb: problem 2101 is visible.
	var nsbProblems []api.ProblemOutput
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+viper.GetString("admin-token")).
		SetHeader("X-Namespace", nsB).
		SetResult(&nsbProblems).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("get problems (nsb) failed: %v %s", err, resp)
	}
	if !containsProblemId(nsbProblems, 2101) {
		t.Fatalf("expected problem 2101 in %s, got %+v", nsB, nsbProblems)
	}

	// In default: problem 2101 must NOT be visible.
	var defaultProblems []api.ProblemOutput
	resp, err = client.R().
		SetHeader("Authorization", "Bearer "+viper.GetString("admin-token")).
		SetHeader("X-Namespace", "default").
		SetResult(&defaultProblems).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("get problems (default) failed: %v %s", err, resp)
	}
	if containsProblemId(defaultProblems, 2101) {
		t.Fatalf("problem 2101 leaked into default namespace")
	}
}

func Test_21_5_Scoreboard_per_namespace(t *testing.T) {
	for _, ns := range []string{"default", nsB} {
		var board []models.ScoreboardEntry
		resp, err := client.R().
			SetHeader("Authorization", "Bearer "+viper.GetString("admin-token")).
			SetHeader("X-Namespace", ns).
			SetResult(&board).
			Get("/api/scores/sum")
		if err != nil || resp.StatusCode() != http.StatusOK {
			t.Fatalf("score sum (%s) failed: %v %s", ns, err, resp)
		}
		t.Logf("score sum[%s]: %d entries", ns, len(board))
	}
}

func containsProblemId(problems []api.ProblemOutput, id int64) bool {
	for _, p := range problems {
		if p.ProblemId == id {
			return true
		}
	}
	return false
}

// adminReq returns a request authenticated with the global admin token.
func adminReq() *resty.Request {
	return client.R().
		SetHeader("Authorization", "Bearer "+viper.GetString("admin-token"))
}

// loginPlain logs in a plain user and returns a client carrying its session.
func loginPlain(t *testing.T, username, password string) *resty.Client {
	t.Helper()
	c := resty.New()
	c.SetBaseURL("http://localhost:8080")
	c.SetRedirectPolicy(resty.NoRedirectPolicy())
	c.OnSuccess(func(_ *resty.Client, r *resty.Response) {
		for _, v := range r.Header().Values("Set-Cookie") {
			if cookie, err := http.ParseSetCookie(v); err == nil {
				c.SetCookie(cookie)
			}
		}
	})

	resp, err := c.R().
		SetBody(map[string]string{"username": username, "password": password}).
		Post("/auth/login/plain")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("login %s failed: %v (status %d) %s",
			username, err, resp.StatusCode(), resp.String())
	}
	return c
}

func Test_21_6_Create_namespace_users(t *testing.T) {
	stdin := strings.NewReader(`---
- username: '` + nsAdminUser + `'
  password: '` + nsAdminPass + `'
- username: '` + nsUserUser + `'
  password: '` + nsUserPass + `'
`)
	cmd := exec.Command(
		"judge", "admin", "import", "users", "plain",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")
	cmd.Stdin = stdin
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to import namespace users: %v\n%s", err, out.String())
	}

	// Resolve their ids.
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
		case nsAdminUser:
			test21NsAdminId = u.Id
		case nsUserUser:
			test21NsUserId = u.Id
		}
	}
	if test21NsAdminId == 0 || test21NsUserId == 0 {
		t.Fatalf("namespace users not found (admin=%d user=%d)",
			test21NsAdminId, test21NsUserId)
	}
}

func Test_21_7_Add_users_to_nsb(t *testing.T) {
	// nsb-admin: admin of nsb. nsb-user: plain member of nsb. Both are taken
	// out of the default namespace below, so they are namespace-scoped only.
	resp, err := adminReq().
		SetHeader("X-Namespace", nsB).
		SetBody(map[string]interface{}{
			"userIds": []int64{test21NsAdminId},
			"isAdmin": true,
		}).
		Put("/api/admin/namespace/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("add nsb admin failed: %v %s", err, resp)
	}

	resp, err = adminReq().
		SetHeader("X-Namespace", nsB).
		SetBody(map[string]interface{}{
			"userIds": []int64{test21NsUserId},
			"isAdmin": false,
		}).
		Put("/api/admin/namespace/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("add nsb user failed: %v %s", err, resp)
	}

	removeFromDefault(t, test21NsAdminId, test21NsUserId)
}

func Test_21_8_Namespace_user_isolation(t *testing.T) {
	c := loginPlain(t, nsUserUser, nsUserPass)

	// Member of nsb: can list nsb problems.
	var nsbProblems []api.ProblemOutput
	resp, err := c.R().
		SetHeader("X-Namespace", nsB).
		SetResult(&nsbProblems).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("nsb-user get problems (nsb) failed: %v %s", err, resp)
	}
	if !containsProblemId(nsbProblems, 2101) {
		t.Fatalf("nsb-user should see problem 2101 in %s", nsB)
	}

	// Not a member of default: must be forbidden.
	resp, err = c.R().
		SetHeader("X-Namespace", "default").
		Get("/api/problems")
	if err != nil {
		t.Fatalf("nsb-user get problems (default) errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("nsb-user reading default namespace: expected 403, got %d %s",
			resp.StatusCode(), resp.String())
	}

	// A plain member cannot manage namespace membership.
	resp, err = c.R().
		SetHeader("X-Namespace", nsB).
		SetBody(map[string]interface{}{
			"userIds": []int64{test21NsUserId},
			"isAdmin": true,
		}).
		Put("/api/admin/namespace/users")
	if err != nil {
		t.Fatalf("nsb-user manage members errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("nsb-user managing members: expected 403, got %d",
			resp.StatusCode())
	}
}

func Test_21_9_Namespace_admin_scope(t *testing.T) {
	c := loginPlain(t, nsAdminUser, nsAdminPass)

	// Admin of nsb: can manage nsb membership.
	resp, err := c.R().
		SetHeader("X-Namespace", nsB).
		SetBody(map[string]interface{}{
			"userIds": []int64{test21NsUserId},
			"isAdmin": false,
		}).
		Put("/api/admin/namespace/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("nsb-admin manage members (nsb) failed: %v %s", err, resp)
	}

	// Not an admin of default: forbidden there.
	resp, err = c.R().
		SetHeader("X-Namespace", "default").
		SetBody(map[string]interface{}{
			"userIds": []int64{test21NsUserId},
			"isAdmin": false,
		}).
		Put("/api/admin/namespace/users")
	if err != nil {
		t.Fatalf("nsb-admin manage members (default) errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("nsb-admin managing default members: expected 403, got %d",
			resp.StatusCode())
	}

	// Namespace admin is not a global admin: cannot create namespaces.
	resp, err = c.R().
		SetBody(map[string]interface{}{"name": "nsc", "title": "Namespace C"}).
		Post("/api/admin/namespace")
	if err != nil {
		t.Fatalf("nsb-admin create namespace errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("nsb-admin creating namespace: expected 403, got %d",
			resp.StatusCode())
	}
}

func Test_21_9_1_Me_lists_namespaces(t *testing.T) {
	c := loginPlain(t, nsUserUser, nsUserPass)

	var me api.MeOutput
	resp, err := c.R().SetResult(&me).Get("/api/me")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("nsb-user get /api/me failed: %v %s", err, resp)
	}

	if len(me.Namespaces) != 1 || me.Namespaces[0].Name != nsB {
		t.Fatalf("expected only %s in /api/me namespaces, got %+v",
			nsB, me.Namespaces)
	}
	if me.Namespaces[0].IsAdmin {
		t.Fatalf("nsb-user should not be admin of %s", nsB)
	}

	c = loginPlain(t, nsAdminUser, nsAdminPass)
	me = api.MeOutput{}
	resp, err = c.R().SetResult(&me).Get("/api/me")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("nsb-admin get /api/me failed: %v %s", err, resp)
	}
	if len(me.Namespaces) != 1 || !me.Namespaces[0].IsAdmin {
		t.Fatalf("expected nsb-admin to be admin of %s, got %+v",
			nsB, me.Namespaces)
	}
}

func Test_21_10_Multi_namespace_read(t *testing.T) {
	// Import a distinct problem (2102) into the default namespace.
	cmd := exec.Command(
		"judge", "admin", "import", "problems",
		"--problems-dir", "resources/22/problems",
		"--testcases-dir", "resources/22/testcases",
		"--namespace", "default",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("import into default failed: %v\n%s", err, stderr.String())
	}
	var sha1 string
	for line := range strings.Lines(stdout.String()) {
		if fields := strings.Fields(line); len(fields) > 0 {
			sha1 = fields[0]
			break
		}
	}
	enable := exec.Command(
		"judge", "admin", "enable", "problem", sha1,
		"--namespace", "default",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")
	if err := enable.Run(); err != nil {
		t.Fatalf("enable in default failed: %v", err)
	}

	// Make nsb-user a member of default as well, so it spans two namespaces.
	resp, err := adminReq().
		SetHeader("X-Namespace", "default").
		SetBody(map[string]interface{}{
			"userIds": []int64{test21NsUserId},
			"isAdmin": false,
		}).
		Put("/api/admin/namespace/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("add nsb-user to default failed: %v %s", err, resp)
	}

	// Re-login to pick up the new membership in the access token.
	c := loginPlain(t, nsUserUser, nsUserPass)

	// Explicit multi-namespace list: both problems are visible.
	var both []api.ProblemOutput
	resp, err = c.R().
		SetHeader("X-Namespace", "default,"+nsB).
		SetResult(&both).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("multi-namespace get problems failed: %v %s", err, resp)
	}
	if !containsProblemId(both, 2101) || !containsProblemId(both, 2102) {
		t.Fatalf("expected both 2101 (nsb) and 2102 (default), got %+v", both)
	}

	// "*" derives all of the caller's namespaces from the JWT.
	var star []api.ProblemOutput
	resp, err = c.R().
		SetHeader("X-Namespace", "*").
		SetResult(&star).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("wildcard get problems failed: %v %s", err, resp)
	}
	if !containsProblemId(star, 2101) || !containsProblemId(star, 2102) {
		t.Fatalf("wildcard expected both 2101 and 2102, got %+v", star)
	}

	// Single namespace still scopes to just that namespace.
	var nsbOnly []api.ProblemOutput
	resp, err = c.R().
		SetHeader("X-Namespace", nsB).
		SetResult(&nsbOnly).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("nsb-only get problems failed: %v %s", err, resp)
	}
	if !containsProblemId(nsbOnly, 2101) || containsProblemId(nsbOnly, 2102) {
		t.Fatalf("nsb-only should have 2101 but not 2102, got %+v", nsbOnly)
	}
}

func Test_21_11_Remove_namespace_user(t *testing.T) {
	// Remove nsb-user from nsb via DELETE.
	resp, err := adminReq().
		SetHeader("X-Namespace", nsB).
		SetBody(map[string]interface{}{
			"userIds": []int64{test21NsUserId},
		}).
		Delete("/api/admin/namespace/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("delete nsb member failed: %v %s", err, resp)
	}

	// After removal (re-login to refresh membership), nsb is no longer readable.
	c := loginPlain(t, nsUserUser, nsUserPass)
	resp, err = c.R().
		SetHeader("X-Namespace", nsB).
		Get("/api/problems")
	if err != nil {
		t.Fatalf("removed user get problems errored: %v", err)
	}
	if resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("removed user reading nsb: expected 403, got %d %s",
			resp.StatusCode(), resp.String())
	}
}

// test21SeedUser is a plain user left behind after the suite, for logging into
// the frontend by hand. The password is always "<username>-pass".
type test21SeedUser struct {
	username      string
	isGlobalAdmin bool
	memberOf      []string // namespace names joined as a plain member
	adminOf       []string // namespace names joined as admin
	wantAdmins    int      // namespaces /api/me should report isAdmin for; -1 = all
}

const nsDisabled = "nsd"

var test21SeedUsers = []test21SeedUser{
	// Switches between two namespaces as a plain member.
	{username: "ns-multi", memberOf: []string{"default", nsB}},
	// Namespace admin of nsb, plain member of default.
	{username: "ns-mixed-admin", memberOf: []string{"default"}, adminOf: []string{nsB}, wantAdmins: 1},
	// Admin of both namespaces.
	{username: "ns-double-admin", adminOf: []string{"default", nsB}, wantAdmins: 2},
	// Member of a disabled namespace, which shows up but cannot be selected.
	{username: "ns-disabled-member", memberOf: []string{nsB, nsDisabled}},
	// Global admin with no explicit membership: sees and administers every
	// namespace, including the disabled one.
	{username: "ns-global-admin", isGlobalAdmin: true, wantAdmins: -1},
}

func Test_21_12_Seed_login_users(t *testing.T) {
	// A namespace that ends up disabled, for the disabled-member user.
	resp, err := adminReq().
		SetBody(map[string]interface{}{
			"name":  nsDisabled,
			"title": "Namespace D (disabled)",
		}).
		Post("/api/admin/namespace")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("create %s failed: %v %s", nsDisabled, err, resp)
	}

	var yaml strings.Builder
	yaml.WriteString("---\n")
	for _, u := range test21SeedUsers {
		yaml.WriteString("- username: '" + u.username + "'\n")
		yaml.WriteString("  password: '" + u.username + "-pass'\n")
		if u.isGlobalAdmin {
			yaml.WriteString("  isAdmin: true\n")
		}
	}
	runJudgeStdin(t, yaml.String(),
		"admin", "import", "users", "plain",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var users []struct {
		Id       int64  `json:"id"`
		Username string `json:"username"`
	}
	resp, err = adminReq().SetResult(&users).Get("/api/admin/users")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("get users failed: %v %s", err, resp)
	}
	userIds := make(map[string]int64, len(users))
	for _, u := range users {
		userIds[u.Username] = u.Id
	}

	addMember := func(namespace string, userId int64, isAdmin bool) {
		t.Helper()
		resp, err := adminReq().
			SetHeader("X-Namespace", namespace).
			SetBody(map[string]interface{}{
				"userIds": []int64{userId},
				"isAdmin": isAdmin,
			}).
			Put("/api/admin/namespace/users")
		if err != nil || resp.StatusCode() != http.StatusOK {
			t.Fatalf("add user %d to %s failed: %v %s", userId, namespace, err, resp)
		}
	}
	for _, u := range test21SeedUsers {
		id, ok := userIds[u.username]
		if !ok {
			t.Fatalf("seed user %s not found", u.username)
		}
		for _, ns := range u.memberOf {
			addMember(ns, id, false)
		}
		for _, ns := range u.adminOf {
			addMember(ns, id, true)
		}
		// Every user starts in default; keep it only where listed.
		if !u.isGlobalAdmin && !slices.Contains(u.memberOf, "default") &&
			!slices.Contains(u.adminOf, "default") {
			removeFromDefault(t, id)
		}
	}

	// Disable nsd only after its members are added.
	var namespaces []models.Namespace
	resp, err = adminReq().SetResult(&namespaces).Get("/api/admin/namespaces")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("list namespaces failed: %v %s", err, resp)
	}
	var nsDisabledId int64
	for _, ns := range namespaces {
		if ns.Name == nsDisabled {
			nsDisabledId = ns.Id
		}
	}
	resp, err = adminReq().
		SetBody(api.PutNamespaceStateInput{Id: nsDisabledId, IsEnabled: false}).
		Put("/api/admin/namespace/state")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("disable %s failed: %v %s", nsDisabled, err, resp)
	}

	// Every seed user can log in and sees exactly its memberships.
	for _, u := range test21SeedUsers {
		c := loginPlain(t, u.username, u.username+"-pass")

		var me api.MeOutput
		resp, err := c.R().SetResult(&me).Get("/api/me")
		if err != nil || resp.StatusCode() != http.StatusOK {
			t.Fatalf("%s get /api/me failed: %v %s", u.username, err, resp)
		}
		want := len(u.memberOf) + len(u.adminOf)
		if u.isGlobalAdmin {
			want = len(namespaces)
		}
		if len(me.Namespaces) != want {
			t.Fatalf("%s: expected %d namespaces, got %+v",
				u.username, want, me.Namespaces)
		}
		admins := 0
		for _, ns := range me.Namespaces {
			if ns.IsAdmin {
				admins++
			}
		}
		wantAdmins := u.wantAdmins
		if wantAdmins == -1 {
			wantAdmins = want
		}
		if admins != wantAdmins {
			t.Fatalf("%s: expected admin of %d namespaces, got %+v",
				u.username, wantAdmins, me.Namespaces)
		}
	}
}

func Test_21_13_Disabled_namespace_hidden_from_members(t *testing.T) {
	// Give the disabled namespace an enabled problem of its own (a global
	// admin can still manage a disabled namespace).
	out := runJudge(t, "admin", "import", "problems",
		"--problems-dir", "resources/21/nsd-problems",
		"--testcases-dir", "resources/21/testcases",
		"--namespace", nsDisabled,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")
	fields := strings.Fields(out)
	if len(fields) == 0 {
		t.Fatalf("no sha1 returned from import: %q", out)
	}
	runJudge(t, "admin", "enable", "problem", fields[0],
		"--namespace", nsDisabled,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	// ns-disabled-member belongs to nsb and the disabled nsd.
	c := loginPlain(t, "ns-disabled-member", "ns-disabled-member-pass")

	// Naming the disabled namespace is refused.
	resp, err := c.R().
		SetHeader("X-Namespace", nsDisabled).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusForbidden {
		t.Fatalf("expected 403 for disabled namespace, got: %v %s", err, resp)
	}

	// Deriving namespaces from membership skips it too: nsb's problem is
	// visible, the disabled namespace's is not.
	var star []api.ProblemOutput
	resp, err = c.R().
		SetHeader("X-Namespace", "*").
		SetResult(&star).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("wildcard get problems failed: %v %s", err, resp)
	}
	if !containsProblemId(star, 2101) {
		t.Fatalf("wildcard expected nsb's 2101, got %+v", star)
	}
	if containsProblemId(star, 2103) {
		t.Fatalf("wildcard leaked disabled namespace's 2103: %+v", star)
	}
}
