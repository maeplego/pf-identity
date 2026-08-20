package org

import (
	"context"
	"strings"
	"time"

	"github.com/portfolio/pf-identity-server/internal/clock"
	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/id"
)

// Service manages organizations (tenant boundaries).
type Service struct {
	Repos domain.Repos
	Clock clock.Clock
}

// Create creates an organization and makes the actor owner.
func (s *Service) Create(ctx context.Context, actorID, name string) (domain.Organization, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 {
		return domain.Organization{}, domain.ErrInvalid
	}
	now := s.Clock.Now().UTC()
	org := domain.Organization{ID: id.New(), Name: name, CreatedAt: now}
	if err := s.Repos.CreateOrganization(ctx, org); err != nil {
		return domain.Organization{}, err
	}
	if err := s.Repos.AddOrganizationMember(ctx, domain.OrganizationMembership{
		OrgID: org.ID, UserID: actorID, Role: domain.OrgRoleOwner, JoinedAt: now,
	}); err != nil {
		return domain.Organization{}, err
	}
	return org, nil
}

// ListForUser returns organizations the user belongs to.
func (s *Service) ListForUser(ctx context.Context, userID string) ([]domain.Organization, error) {
	return s.Repos.ListOrganizationsForUser(ctx, userID)
}

// ListMembers returns org members (with email/name) when actor is a member.
func (s *Service) ListMembers(ctx context.Context, actorID, orgID string) ([]domain.OrgMemberDetail, error) {
	if _, err := s.Repos.GetOrganizationMembership(ctx, orgID, actorID); err != nil {
		return nil, domain.ErrForbidden
	}
	members, err := s.Repos.ListOrganizationMembers(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.OrgMemberDetail, 0, len(members))
	for _, m := range members {
		detail := domain.OrgMemberDetail{
			UserID: m.UserID, Role: string(m.Role), JoinedAt: m.JoinedAt,
		}
		if u, err := s.Repos.GetByID(ctx, m.UserID); err == nil {
			detail.Email = u.Email
			detail.Name = u.Name
		}
		out = append(out, detail)
	}
	return out, nil
}

// SetActiveOrg stores the user's active tenant on their browser session (cookie path).
func (s *Service) SetActiveOrg(ctx context.Context, userID, sessionTokenHash, orgID string) error {
	if _, err := s.Repos.GetOrganizationMembership(ctx, orgID, userID); err != nil {
		return domain.ErrForbidden
	}
	return s.Repos.SetSessionActiveOrg(ctx, sessionTokenHash, orgID)
}

// SetActiveOrgBySID stores active org for an OP session identified by sid (Bearer / refresh path).
func (s *Service) SetActiveOrgBySID(ctx context.Context, userID, sid, orgID string) error {
	if _, err := s.Repos.GetOrganizationMembership(ctx, orgID, userID); err != nil {
		return domain.ErrForbidden
	}
	sess, err := s.Repos.GetSessionBySID(ctx, sid)
	if err != nil {
		return domain.ErrNotFound
	}
	if sess.UserID != userID {
		return domain.ErrForbidden
	}
	return s.Repos.SetSessionActiveOrgBySID(ctx, sid, orgID)
}

// MembershipViews lists org memberships with names for userinfo.
func MembershipViews(ctx context.Context, repos domain.Repos, userID string) ([]domain.OrgMembershipView, error) {
	orgs, err := repos.ListOrganizationsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	var out []domain.OrgMembershipView
	for _, org := range orgs {
		m, err := repos.GetOrganizationMembership(ctx, org.ID, userID)
		if err != nil {
			continue
		}
		out = append(out, domain.OrgMembershipView{OrgID: org.ID, OrgName: org.Name, Role: string(m.Role)})
	}
	return out, nil
}

// PrimaryMembership picks active org from session SID or first membership.
type PrimaryResolver interface {
	ResolvePrimaryOrg(ctx context.Context, userID, sessionSID string) (domain.OrgMembershipView, []domain.OrgMembershipView, error)
}

// NowUTC helper for tests.
func NowUTC(c clock.Clock) time.Time { return c.Now().UTC() }
