package postgres

import (
	"context"
	"strings"

	"github.com/portfolio/pf-identity-server/internal/domain"
)

func (s *Store) CreateOrganization(ctx context.Context, org domain.Organization) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO organizations (id, name, created_at) VALUES ($1, $2, $3)`,
		org.ID, org.Name, org.CreatedAt.UTC(),
	)
	return mapErr(err)
}

func (s *Store) GetOrganization(ctx context.Context, id string) (domain.Organization, error) {
	var org domain.Organization
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, created_at FROM organizations WHERE id = $1`, id,
	).Scan(&org.ID, &org.Name, &org.CreatedAt)
	if err != nil {
		return domain.Organization{}, mapErr(err)
	}
	return org, nil
}

func (s *Store) ListOrganizationsForUser(ctx context.Context, userID string) ([]domain.Organization, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT o.id, o.name, o.created_at
		FROM organizations o
		JOIN organization_memberships m ON m.org_id = o.id
		WHERE m.user_id = $1
		ORDER BY o.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Organization
	for rows.Next() {
		var org domain.Organization
		if err := rows.Scan(&org.ID, &org.Name, &org.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, org)
	}
	return out, rows.Err()
}

func (s *Store) AddOrganizationMember(ctx context.Context, m domain.OrganizationMembership) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO organization_memberships (org_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4)`,
		m.OrgID, m.UserID, string(m.Role), m.JoinedAt.UTC(),
	)
	return mapErr(err)
}

func (s *Store) ListOrganizationMembers(ctx context.Context, orgID string) ([]domain.OrganizationMembership, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT org_id, user_id, role, joined_at
		FROM organization_memberships WHERE org_id = $1 ORDER BY joined_at`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.OrganizationMembership
	for rows.Next() {
		var m domain.OrganizationMembership
		var role string
		if err := rows.Scan(&m.OrgID, &m.UserID, &role, &m.JoinedAt); err != nil {
			return nil, err
		}
		m.Role = domain.OrgRole(role)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetOrganizationMembership(ctx context.Context, orgID, userID string) (domain.OrganizationMembership, error) {
	var m domain.OrganizationMembership
	var role string
	err := s.pool.QueryRow(ctx, `
		SELECT org_id, user_id, role, joined_at
		FROM organization_memberships WHERE org_id = $1 AND user_id = $2`, orgID, userID,
	).Scan(&m.OrgID, &m.UserID, &role, &m.JoinedAt)
	if err != nil {
		return domain.OrganizationMembership{}, mapErr(err)
	}
	m.Role = domain.OrgRole(role)
	return m, nil
}

func (s *Store) SetSessionActiveOrg(ctx context.Context, tokenHash, orgID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE sessions SET active_org_id = $2 WHERE token_hash = $1`, tokenHash, orgID)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) SetSessionActiveOrgBySID(ctx context.Context, sid, orgID string) error {
	if strings.TrimSpace(sid) == "" {
		return domain.ErrInvalid
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE sessions SET active_org_id = $2 WHERE sid = $1`, sid, orgID)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
