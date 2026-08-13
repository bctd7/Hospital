package patient

import (
	"testing"

	"github.com/google/uuid"

	"hospital/common/authn"
)

func TestManagerAcceptsPatientIdentityWithoutStaffRole(t *testing.T) {
	err := requirePatient(authn.Principal{AccountID: uuid.NewString(), AccountType: authn.AccountTypePatient})
	if err != nil {
		t.Fatalf("patient read rejected: %v", err)
	}
}
