package services

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type PermissionService interface {
	HasPermission(ctx context.Context, userID, method, path string) (bool, error)
}

type permissionService struct {
	db *sqlx.DB
}

func NewPermissionService(db *sqlx.DB) PermissionService {
	return &permissionService{db: db}
}

// HasPermission looks up the member's role, then checks whether any
// permission row for that role matches the requested method and path.
//
// userID here is expected to be a member_id (the value embedded in the
// JWT's "user_id" claim at token-issuance time), not the members table's
// internal serial "id" column.
func (p *permissionService) HasPermission(ctx context.Context, userID, method, path string) (bool, error) {
	if userID == "" {
		return false, fmt.Errorf("permission check requires a userID")
	}

	var role string
	err := p.db.GetContext(ctx, &role,
		`SELECT role FROM members WHERE member_id = $1 AND deleted = FALSE`,
		userID,
	)
	if err != nil {
		return false, fmt.Errorf("looking up member role: %w", err)
	}

	var count int
	err = p.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM role_permissions
		 WHERE role = $1
		   AND method = $2
		   AND $3 LIKE path_pattern`,
		role, method, path,
	)
	if err != nil {
		return false, fmt.Errorf("checking role permissions: %w", err)
	}

	return count > 0, nil
}
