package events

const EventTypeIdentityAuthorizationChangedV1 = "identity.authorization.changed.v1"

type IdentityAuthorizationChangedV1Payload struct {
	OperationID          string `json:"operation_id"`
	Action               string `json:"action"`
	AuthorizationVersion int64  `json:"authorization_version"`
}
