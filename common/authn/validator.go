package authn

import "context"

// PrincipalValidator performs a runtime validity check after token signature
// verification. Authorization-version validation is implemented by authz/version.
type PrincipalValidator interface {
	ValidatePrincipal(ctx context.Context, principal Principal) error
}
