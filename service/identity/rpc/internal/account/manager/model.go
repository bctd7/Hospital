package manager

import "time"

const (
	IdentityTypePatient    = "patient"
	IdentityTypeDoctor     = "doctor"
	IdentityTypeSuperAdmin = "super_admin"

	StaffStatusActive  = "active"
	StaffStatusRevoked = "revoked"

	ActionPromoteDoctor          = "identity.doctor.promoted"
	ActionUpdateDoctor           = "identity.doctor.updated"
	ActionChangeDoctorDepartment = "identity.doctor.department_changed"
	ActionRevokeDoctor           = "identity.doctor.revoked"
	ActionDisableAccount         = "identity.account.disabled"
	ActionEnableAccount          = "identity.account.enabled"
)

type DisplayProfile struct {
	Nickname          string
	ManagementVersion int64
}

type DoctorSummary struct {
	AccountID         string
	DisplayName       string
	DepartmentID      string
	AvatarURL         string
	Description       string
	ManagementVersion int64
}

type DoctorPage struct {
	Items    []DoctorSummary
	Page     int64
	PageSize int64
	Total    int64
}

type Account struct {
	ID                      string
	Nickname                string
	DisplayName             string
	AvatarURL               string
	MaskedPhone             string
	AccountStatus           string
	AccountType             string
	StaffStatus             string
	DepartmentID            string
	DepartmentName          string
	ManagementVersion       int64
	PhoneVerificationStatus string
	PhoneVerificationSource string
	StaffNo                 string
	Description             string
	Roles                   []string
	AuthorizationVersion    int64
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (a Account) IdentityType() string {
	for _, role := range a.Roles {
		if role == IdentityTypeSuperAdmin {
			return IdentityTypeSuperAdmin
		}
	}
	for _, role := range a.Roles {
		if role == "department_doctor" && a.StaffStatus == StaffStatusActive {
			return IdentityTypeDoctor
		}
	}
	return IdentityTypePatient
}

type AccountPage struct {
	Items    []Account
	Page     int64
	PageSize int64
	Total    int64
}

type AccountFilter struct {
	Page         int64
	PageSize     int64
	Nickname     string
	IdentityType string
	Status       string
	DepartmentID string
}

type DoctorProfileInput struct {
	DisplayName string
	StaffNo     string
	AvatarURL   string
	Description string
}

type OptionalDoctorProfileInput struct {
	DisplayName *string
	StaffNo     *string
	AvatarURL   *string
	Description *string
}

type Operation struct {
	OperatorAccountID string
	TargetAccountID   string
	Action            string
}

type Change struct {
	OperationID       string
	OperatorAccountID string
	TargetAccountID   string
	Action            string
	Before            Account
	After             Account
	RequestID         string
}
