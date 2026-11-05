package interfaces

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type NamespaceStorage interface {
	CreateNamespace(
		ctx context.Context,
		ns *models.Namespace,
	) error

	GetNamespaces(
		ctx context.Context,
	) ([]models.Namespace, error)

	GetNamespace(
		ctx context.Context,
		id int64,
	) (models.Namespace, error)

	GetNamespaceByName(
		ctx context.Context,
		name string,
	) (models.Namespace, error)

	UpdateNamespaceState(
		ctx context.Context,
		id int64,
		isEnabled bool,
	) error

	// ImportNamespaceUsers creates the users whose username doesn't exist yet
	// and makes every user a member of the namespace with its namespace admin
	// flag, all in one transaction. An existing user's account (credential,
	// args, global admin flag) is never modified - only its membership.
	ImportNamespaceUsers(
		ctx context.Context,
		namespaceId int64,
		users []models.NamespaceUserImport,
	) ([]models.NamespaceUserImported, error)

	// UpsertNamespaceMembers adds (or updates the admin flag of) the given users
	// in a namespace.
	UpsertNamespaceMembers(
		ctx context.Context,
		namespaceId int64,
		userIds []int64,
		isAdmin bool,
	) error

	RemoveNamespaceMembers(
		ctx context.Context,
		namespaceId int64,
		userIds []int64,
	) error

	GetNamespaceMembers(
		ctx context.Context,
		namespaceId int64,
	) ([]models.NamespaceMember, error)

	// GetUserNamespaces returns the namespaces a user belongs to and the subset
	// where they are admin.
	GetUserNamespaces(
		ctx context.Context,
		userId int64,
	) (namespaces []int64, adminIn []int64, err error)
}
