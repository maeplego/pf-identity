package memory

import (
	"context"
	"sort"

	"github.com/portfolio/pf-identity-server/internal/domain"
)

func (s *Store) CreateOrganization(_ context.Context, org domain.Organization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.orgs == nil {
		s.orgs = map[string]domain.Organization{}
	}
	s.orgs[org.ID] = org
	return nil
}

func (s *Store) GetOrganization(_ context.Context, id string) (domain.Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	org, ok := s.orgs[id]
	if !ok {
		return domain.Organization{}, domain.ErrNotFound
	}
	return org, nil
}

func (s *Store) ListOrganizationsForUser(_ context.Context, userID string) ([]domain.Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.Organization
	for orgID, members := range s.orgMembers {
		if _, ok := members[userID]; ok {
			if org, ok := s.orgs[orgID]; ok {
				out = append(out, org)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *Store) AddOrganizationMember(_ context.Context, m domain.OrganizationMembership) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.orgMembers[m.OrgID] == nil {
		s.orgMembers[m.OrgID] = map[string]domain.OrganizationMembership{}
	}
	if _, exists := s.orgMembers[m.OrgID][m.UserID]; exists {
		return domain.ErrConflict
	}
	s.orgMembers[m.OrgID][m.UserID] = m
	return nil
}

func (s *Store) ListOrganizationMembers(_ context.Context, orgID string) ([]domain.OrganizationMembership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	members, ok := s.orgMembers[orgID]
	if !ok {
		return []domain.OrganizationMembership{}, nil
	}
	out := make([]domain.OrganizationMembership, 0, len(members))
	for _, m := range members {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].JoinedAt.Before(out[j].JoinedAt) })
	return out, nil
}

func (s *Store) GetOrganizationMembership(_ context.Context, orgID, userID string) (domain.OrganizationMembership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	members, ok := s.orgMembers[orgID]
	if !ok {
		return domain.OrganizationMembership{}, domain.ErrNotFound
	}
	m, ok := members[userID]
	if !ok {
		return domain.OrganizationMembership{}, domain.ErrNotFound
	}
	return m, nil
}

func (s *Store) SetSessionActiveOrg(_ context.Context, tokenHash, orgID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[tokenHash]
	if !ok {
		return domain.ErrNotFound
	}
	sess.ActiveOrgID = orgID
	s.sessions[tokenHash] = sess
	return nil
}
