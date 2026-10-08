package models

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/uptrace/bun"
)

// DefaultNamespaceId is the namespace existing single-tenant data is backfilled
// into and the fallback target when a request omits X-Namespace.
const DefaultNamespaceId int64 = 1

// refSHA1 hashes a problem/tool payload together with its namespace, so the
// same content imported into two namespaces gets two distinct sha1s (sha1 is
// the problem/tool primary key). The default namespace hashes the payload
// alone, keeping the sha1s of everything imported before namespaces existed.
func refSHA1(payload []byte, namespaceId int64) string {
	h := sha1.New()
	if namespaceId != 0 && namespaceId != DefaultNamespaceId {
		fmt.Fprintf(h, "namespace:%d\n", namespaceId)
	}
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// Namespace access levels used by the CheckNamespace middleware.
type NamespaceAccess = int

const (
	NamespaceRead  NamespaceAccess = iota // any member of the namespace
	NamespaceAdmin                        // admin of the namespace (or global admin)
)

type Namespace struct {
	bun.BaseModel `json:"-" yaml:"-" bun:"table:namespace"`

	Id    int64  `json:"id"    yaml:"id"    bun:"id,pk,autoincrement"`
	Name  string `json:"name"  yaml:"name"  bun:"name,unique"`
	Title string `json:"title" yaml:"title" bun:"title"`

	// Args are default job args for every submission in this namespace -
	// the outermost, most general layer of the same overlay problem/stage/
	// testcase args already use (see utils.ArgsOverlay), so a problem can
	// still override a namespace default for the same key.
	Args      map[string]string `json:"args"       yaml:"args"       bun:"args"`
	CreatedAt int64             `json:"createdAt"  yaml:"createdAt"  bun:"created_at"`
	IsEnabled bool              `json:"isEnabled"  yaml:"isEnabled"  bun:"is_enabled"`
}

// UserNamespace is the membership join between users and namespaces, with the
// per-namespace admin role.
type UserNamespace struct {
	bun.BaseModel `json:"-" yaml:"-" bun:"table:user_namespace"`

	UserId      int64 `json:"userId"      yaml:"userId"      bun:"user_id,pk"`
	NamespaceId int64 `json:"namespaceId" yaml:"namespaceId" bun:"namespace_id,pk"`
	IsAdmin     bool  `json:"isAdmin"     yaml:"isAdmin"     bun:"is_admin"`
}

// NamespaceMember is a namespace membership with the member's username, so
// namespace admins (who can't list the global user directory) can tell
// members apart.
type NamespaceMember struct {
	UserId      int64  `json:"userId"      yaml:"userId"      bun:"user_id"`
	NamespaceId int64  `json:"namespaceId" yaml:"namespaceId" bun:"namespace_id"`
	Username    string `json:"username"    yaml:"username"    bun:"username"`
	IsAdmin     bool   `json:"isAdmin"     yaml:"isAdmin"     bun:"is_admin"`
}

// NamespaceUserImport is one user a namespace admin imports: created if its
// username is new, then made a member of the namespace.
type NamespaceUserImport struct {
	User UserResource
	// Cred is the plain-login credential for a newly created user; nil for
	// users that log in another way (e.g. CSSSO).
	Cred *PlainCred
	// IsAdmin is the namespace admin flag, never the global one.
	IsAdmin bool
}

// NamespaceUserImported reports what importing one user did.
type NamespaceUserImported struct {
	Id       int64  `json:"id" yaml:"id"`
	Username string `json:"username" yaml:"username"`
	// Created is false when the username already existed: that user was
	// only added to the namespace, its account left untouched.
	Created bool `json:"created" yaml:"created"`
	IsAdmin bool `json:"isAdmin" yaml:"isAdmin"`
}
