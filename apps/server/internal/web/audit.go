package web

import (
	"log"
	"net"
	"net/http"

	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/id"
)

func (s *Server) audit(r *http.Request, typ, subject, clientID, note string) {
	err := s.Repos.AppendAudit(r.Context(), domain.AuditEvent{
		ID:       id.New(),
		Type:     typ,
		At:       s.now(),
		Subject:  subject,
		ClientID: clientID,
		IP:       requestIP(r),
		Note:     note,
	})
	if err != nil {
		log.Printf("audit append failed type=%s", typ)
	}
}

func requestIP(r *http.Request) string {
	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = host
	}
	return ip
}
