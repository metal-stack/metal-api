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
	log      *zap.SugaredLogger
	q        *rego.PreparedEvalQuery
	basePath string
}

func (r *regoDecider) newRegoInput(req *http.Request, u *security.User, permissions Permissions) (map[string]interface{}, error) {
	return map[string]interface{}{
		"method": req.Method,
		"path":   strings.Split(strings.TrimPrefix(req.URL.Path, r.basePath), "/"),
		"subject": map[string]interface{}{
			"user":   u.Name,
			"groups": u.Groups,
		},
		"permissions": permissions,
	}, nil
}

func newRegoDecider(log *zap.SugaredLogger, basePath string) (*regoDecider, error) {
	options := []func(r *rego.Rego){
		rego.Query("x = data.api.v1.metalstack.io.authz.allow"),
	}

	files, err := v1.RegoPolicies.ReadDir("policies")
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		data, err := v1.RegoPolicies.ReadFile("policies/" + f.Name())
		if err != nil {
			return nil, err
		}
		options = append(options, rego.Module(f.Name(), string(data)))
	}

	query, err := rego.New(
		options...,
	).PrepareForEval(context.Background())
	if err != nil {
		return nil, err
	}

	return &regoDecider{
		q:        &query,
		log:      log,
		basePath: basePath,
	}, nil
}

func (r *regoDecider) Decide(ctx context.Context, req *http.Request, u *security.User, permissions Permissions) error {
	input, err := r.newRegoInput(req, u, permissions)
	if err != nil {
		return err
	}

	r.log.Debugw("rego evaluation", "input", input)

	results, err := r.q.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return errors.Wrap(err, "error evaluating rego result set")
	} else if len(results) == 0 {
		return fmt.Errorf("error evaluating rego result set: results have no length")
	} else if allowed, ok := results[0].Bindings["x"].(bool); !ok {
		return fmt.Errorf("error evaluating rego result set: unexpected response type")
	} else {
		r.log.Debugw("made auth decision", "results", results)

		if !allowed {
			return fmt.Errorf("access denied")
		}

		return nil
	}
}
