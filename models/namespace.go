package models

import "github.com/uptrace/bun"

// DefaultNamespaceId is the namespace existing single-tenant data is backfilled
// into and the fallback target when a request omits X-Namespace.
const DefaultNamespaceId int64 = 1

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
