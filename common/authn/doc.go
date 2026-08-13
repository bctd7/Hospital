// Package authn answers "who is calling". It owns Principal, access-token
// verification, and HTTP/gRPC authentication middleware. Permission and scope
// decisions belong to authz; authorization-version storage belongs to
// authz/version.
package authn
