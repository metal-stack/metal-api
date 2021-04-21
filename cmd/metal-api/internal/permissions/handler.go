package permissions

import (
	"context"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/security"
	"go.uber.org/zap"
)

type PermissionsHandler struct {
	log     *zap.SugaredLogger
	decider *regoDecider

	ug security.UserGetter
}

func NewPermissionsHandler(log *zap.SugaredLogger, basePath string, ug security.UserGetter) (*PermissionsHandler, error) {
	d, err := newRegoDecider(log, basePath)
	if err != nil {
		return nil, err
	}

	return &PermissionsHandler{
		decider: d,
		log:     log,
		ug:      ug,
	}, nil
}

// Authz is a go-restful request filter method that will do the request authz for requests with Bearer tokens.
//
// It enriches the request attributes with a resource scope to be used in the datastore.
func (p *PermissionsHandler) Authz(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	u, err := p.ug.User(req.Request)
	if err != nil {
		_ = resp.WriteHeaderAndEntity(http.StatusForbidden, httperrors.Forbidden(err))
		return
	}

	// TODO: How to get permissions? Must be composed from roles!
	permissions := mockRoles(u).MergePermissions() // FIXME

	isAdmin, err := p.decider.Decide(req.Request.Context(), req.Request, u, permissions)
	if err != nil {
		_ = resp.WriteHeaderAndEntity(http.StatusForbidden, httperrors.Forbidden(err))
		return
	}

	scope := datastore.NewResourceScope(
		datastore.Predicates{"*"}, // FIXME // TODO: How to always get all latest tenants (on behalf) here?
		datastore.Predicates{"*"}, // FIXME // TODO: How to always get latest projects here?
		datastore.Predicates{"*"}, // FIXME // TODO: Do we want to go this far? Where to get these?
		isAdmin,
	)
	// scope := datastore.EverythingScope // FIXME

	req.SetAttribute("scope", scope)
	p.log.Debugw("set request attribute", "scope", scope)

	chain.ProcessFilter(req, resp)
}

// ListPermissions lists all permissions handled by the permissions handler
func (p *PermissionsHandler) ListPermissions(ctx context.Context) ([]string, error) {
	return p.decider.ListPermissions(ctx)
}