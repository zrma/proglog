package auth

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func New(model, policy string) (*Authorizer, error) {
	enforcer, err := casbin.NewEnforcer(model, policy)
	if err != nil {
		return nil, err
	}

	return &Authorizer{enforcer: enforcer}, nil
}

type Authorizer struct {
	enforcer *casbin.Enforcer
}

func (a *Authorizer) Authorize(sub, obj, act string) error {
	if ok, err := a.enforcer.Enforce(sub, obj, act); err != nil {
		return err
	} else if !ok {
		msg := fmt.Sprintf("%s not permitted to %s %s", sub, act, obj)
		st := status.New(codes.PermissionDenied, msg)
		return st.Err()
	}

	return nil
}
