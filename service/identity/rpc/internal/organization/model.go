package organization

type UnitType string

const (
	UnitTypeHospital   UnitType = "hospital"
	UnitTypeCampus     UnitType = "campus"
	UnitTypeDepartment UnitType = "department"
)

func (t UnitType) Valid() bool {
	switch t {
	case UnitTypeHospital, UnitTypeCampus, UnitTypeDepartment:
		return true
	default:
		return false
	}
}

func (t UnitType) RequiredParentType() (UnitType, bool) {
	switch t {
	case UnitTypeCampus:
		return UnitTypeHospital, true
	case UnitTypeDepartment:
		return UnitTypeCampus, true
	default:
		return "", false
	}
}

type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
)

func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusDisabled:
		return true
	default:
		return false
	}
}

type Unit struct {
	ID          string
	ParentID    string
	Type        UnitType
	Code        string
	Name        string
	Status      Status
	ChildCount  int64
	DoctorCount int64
	Version     int64
}

type ListFilter struct {
	Type     UnitType
	ParentID *string
	Status   *Status
}

// DirectoryContext is the public, read-only organization entry point.
// Only the active hospital root and its active campuses are included.
type DirectoryContext struct {
	Hospital Unit
	Campuses []Unit
}

// DirectoryDepartment contains the public department fields plus the
// authoritative campus name required by the HTTP directory contract.
type DirectoryDepartment struct {
	Unit
	CampusName string
}
