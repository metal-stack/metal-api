package permissions

import (
	_ "embed"

	"context"
	"fmt"
	"net/http"
	"strings"

	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/security"
	"github.com/open-policy-agent/opa/rego"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// ideas are taken from: https://www.openpolicyagent.org/docs/latest/integration/#integrating-with-the-go-api

type regoDecider struct {
	log          *zap.SugaredLogger
	qDecision    *rego.PreparedEvalQuery
	qPermissions *rego.PreparedEvalQuery
	basePath     string
}

func (r *regoDecider) newRegoInput(req *http.Request, u *security.User, permissions Permissions) map[string]interface{} {
	return map[string]interface{}{
		"method": req.Method,
		"path":   strings.Split(strings.TrimPrefix(req.URL.Path, r.basePath), "/"),
		"subject": map[string]interface{}{
			"user":   u.Name,
			"groups": u.Groups,
		},
		"permissions": permissions,
	}
}

func newRegoDecider(log *zap.SugaredLogger, basePath string) (*regoDecider, error) {
	files, err := v1.RegoPolicies.ReadDir("policies")
	if err != nil {
		return nil, err
	}

	var moduleLoads []func(r *rego.Rego)
	for _, f := range files {
		data, err := v1.RegoPolicies.ReadFile("policies/" + f.Name())
		if err != nil {
			return nil, err
		}
		moduleLoads = append(moduleLoads, rego.Module(f.Name(), string(data)))
	}

	qDecision, err := rego.New(
		append(moduleLoads, rego.Query("x = data.api.v1.metalstack.io.authz.decision"))...,
	).PrepareForEval(context.Background())
	if err != nil {
		return nil, err
	}

	qPermissions, err := rego.New(
		append(moduleLoads, rego.Query("x = data.api.v1.metalstack.io.authz.permissions"))...,
	).PrepareForEval(context.Background())
	if err != nil {
		return nil, err
	}

	return &regoDecider{
		qDecision:    &qDecision,
		qPermissions: &qPermissions,
		log:          log,
		basePath:     basePath,
	}, nil
}

func (r *regoDecider) Decide(ctx context.Context, req *http.Request, u *security.User, permissions Permissions) (bool, error) {
	input := r.newRegoInput(req, u, permissions)

	r.log.Debugw("rego evaluation", "input", input)

	results, err := r.qDecision.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, errors.Wrap(err, "error evaluating rego result set")
	}

	if len(results) == 0 {
		return false, fmt.Errorf("error evaluating rego result set: results have no length")
	}

	decision, ok := results[0].Bindings["x"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("error evaluating rego result set: unexpected response type")
	}

	allow, ok := decision["allow"].(bool)
	if !ok {
		return false, fmt.Errorf("error evaluating rego result set: unexpected response type")
	}

	// TODO remove, only for devel:
	// r.log.Debugw("made auth decision", "results", results)

	if !allow {
		reason, ok := decision["reason"].(string)
		if ok {
			return false, fmt.Errorf("access denied: %s", reason)
		}
		return false, fmt.Errorf("access denied")
	}

	return decision["isAdmin"].(bool), nil
}

func (r *regoDecider) ListPermissions(ctx context.Context) ([]string, error) {
	results, err := r.qPermissions.Eval(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error evaluating rego result set")
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("error evaluating rego result set: results have no length")
	}

	set, ok := results[0].Bindings["x"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("error evaluating rego result set: unexpected response type")
	}

	var ps []string
	for _, p := range set {
		p := p.(string)
		ps = append(ps, p)
	}

	return ps, nil
}
