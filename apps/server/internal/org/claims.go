package org

import (
	"context"

	"github.com/portfolio/pf-identity-server/internal/domain"
)

// PrimaryOrg returns the active org claim and all memberships for userinfo.
func PrimaryOrg(ctx context.Context, repos domain.Repos, userID, sessionSID string) (domain.OrgMembershipView, []domain.OrgMembershipView, error) {
	views, err := MembershipViews(ctx, repos, userID)
	if err != nil {
		return domain.OrgMembershipView{}, nil, err
	}
	if len(views) == 0 {
		return domain.OrgMembershipView{}, views, domain.ErrNotFound
	}
	activeOrgID := ""
	if sessionSID != "" {
		if sess, err := repos.GetSessionBySID(ctx, sessionSID); err == nil {
			activeOrgID = sess.ActiveOrgID
		}
	}
	if activeOrgID != "" {
		for _, v := range views {
			if v.OrgID == activeOrgID {
				return v, views, nil
			}
		}
	}
	return views[0], views, nil
}
