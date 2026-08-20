package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/id"
)

type adminOrgView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type adminOrgMemberView struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

func viewOrg(o domain.Organization) adminOrgView {
	return adminOrgView{
		ID:        o.ID,
		Name:      o.Name,
		CreatedAt: o.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func (s *Server) handleAdminListOrganizations(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	list, err := s.Repos.ListOrganizations(r.Context())
	if err != nil {
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	out := make([]adminOrgView, 0, len(list))
	for _, o := range list {
		out = append(out, viewOrg(o))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAdminCreateOrganization(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var in struct {
		Name         string `json:"name"`
		OwnerUserID  string `json:"owner_user_id"`
		OwnerEmail   string `json:"owner_email"`
	}
	if err := decodeJSON(r, &in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 120 {
		http.Error(w, "name is required (max 120)", http.StatusBadRequest)
		return
	}
	ownerID := strings.TrimSpace(in.OwnerUserID)
	if ownerID == "" {
		email := strings.TrimSpace(strings.ToLower(in.OwnerEmail))
		if email == "" {
			http.Error(w, "owner_user_id or owner_email is required", http.StatusBadRequest)
			return
		}
		u, err := s.Repos.GetByEmail(r.Context(), email)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				http.Error(w, "owner not found", http.StatusNotFound)
				return
			}
			http.Error(w, "store", http.StatusInternalServerError)
			return
		}
		ownerID = u.ID
	} else if _, err := s.Repos.GetByID(r.Context(), ownerID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "owner not found", http.StatusNotFound)
			return
		}
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	now := s.now().UTC()
	org := domain.Organization{ID: id.New(), Name: name, CreatedAt: now}
	if err := s.Repos.CreateOrganization(r.Context(), org); err != nil {
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	if err := s.Repos.AddOrganizationMember(r.Context(), domain.OrganizationMembership{
		OrgID: org.ID, UserID: ownerID, Role: domain.OrgRoleOwner, JoinedAt: now,
	}); err != nil {
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, viewOrg(org))
}

func (s *Server) handleAdminGetOrganization(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	org, err := s.Repos.GetOrganization(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, viewOrg(org))
}

func (s *Server) handleAdminListOrganizationMembers(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	orgID := r.PathValue("id")
	if _, err := s.Repos.GetOrganization(r.Context(), orgID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	members, err := s.Repos.ListOrganizationMembers(r.Context(), orgID)
	if err != nil {
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	out := make([]adminOrgMemberView, 0, len(members))
	for _, m := range members {
		v := adminOrgMemberView{
			UserID:   m.UserID,
			Role:     string(m.Role),
			JoinedAt: m.JoinedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if u, err := s.Repos.GetByID(r.Context(), m.UserID); err == nil {
			v.Email = u.Email
			v.Name = u.Name
		}
		out = append(out, v)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAdminAddOrganizationMember(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	orgID := r.PathValue("id")
	if _, err := s.Repos.GetOrganization(r.Context(), orgID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	var in struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Role   string `json:"role"`
	}
	if err := decodeJSON(r, &in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	role := domain.OrgRole(strings.TrimSpace(in.Role))
	if role == "" {
		role = domain.OrgRoleMember
	}
	if role != domain.OrgRoleOwner && role != domain.OrgRoleMember {
		http.Error(w, "role must be owner or member", http.StatusBadRequest)
		return
	}
	userID := strings.TrimSpace(in.UserID)
	if userID == "" {
		email := strings.TrimSpace(strings.ToLower(in.Email))
		if email == "" {
			http.Error(w, "user_id or email is required", http.StatusBadRequest)
			return
		}
		u, err := s.Repos.GetByEmail(r.Context(), email)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				http.Error(w, "user not found", http.StatusNotFound)
				return
			}
			http.Error(w, "store", http.StatusInternalServerError)
			return
		}
		userID = u.ID
	} else if _, err := s.Repos.GetByID(r.Context(), userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	now := s.now().UTC()
	m := domain.OrganizationMembership{OrgID: orgID, UserID: userID, Role: role, JoinedAt: now}
	if err := s.Repos.AddOrganizationMember(r.Context(), m); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			http.Error(w, "already a member", http.StatusConflict)
			return
		}
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	v := adminOrgMemberView{
		UserID: userID, Role: string(role), JoinedAt: now.Format("2006-01-02T15:04:05Z"),
	}
	if u, err := s.Repos.GetByID(r.Context(), userID); err == nil {
		v.Email = u.Email
		v.Name = u.Name
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) handleAdminUpdateOrganizationMember(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	orgID := r.PathValue("id")
	userID := r.PathValue("userId")
	var in struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(r, &in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	role := domain.OrgRole(strings.TrimSpace(in.Role))
	if role != domain.OrgRoleOwner && role != domain.OrgRoleMember {
		http.Error(w, "role must be owner or member", http.StatusBadRequest)
		return
	}
	current, err := s.Repos.GetOrganizationMembership(r.Context(), orgID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	if current.Role == domain.OrgRoleOwner && role != domain.OrgRoleOwner {
		owners, err := s.countOrgOwnersCtx(r, orgID)
		if err != nil {
			http.Error(w, "store", http.StatusInternalServerError)
			return
		}
		if owners <= 1 {
			http.Error(w, "cannot demote the last owner", http.StatusForbidden)
			return
		}
	}
	if err := s.Repos.UpdateOrganizationMemberRole(r.Context(), orgID, userID, role); err != nil {
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	v := adminOrgMemberView{
		UserID: userID, Role: string(role), JoinedAt: current.JoinedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if u, err := s.Repos.GetByID(r.Context(), userID); err == nil {
		v.Email = u.Email
		v.Name = u.Name
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handleAdminRemoveOrganizationMember(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	orgID := r.PathValue("id")
	userID := r.PathValue("userId")
	current, err := s.Repos.GetOrganizationMembership(r.Context(), orgID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	if current.Role == domain.OrgRoleOwner {
		owners, err := s.countOrgOwnersCtx(r, orgID)
		if err != nil {
			http.Error(w, "store", http.StatusInternalServerError)
			return
		}
		if owners <= 1 {
			http.Error(w, "cannot remove the last owner", http.StatusForbidden)
			return
		}
	}
	if err := s.Repos.RemoveOrganizationMember(r.Context(), orgID, userID); err != nil {
		http.Error(w, "store", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) countOrgOwnersCtx(r *http.Request, orgID string) (int, error) {
	members, err := s.Repos.ListOrganizationMembers(r.Context(), orgID)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, m := range members {
		if m.Role == domain.OrgRoleOwner {
			n++
		}
	}
	return n, nil
}
