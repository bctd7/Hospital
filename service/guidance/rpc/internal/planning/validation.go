package planning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"hospital/common/authn"

	"github.com/google/uuid"
)

func normalizeGenerate(command GenerateCommand) ([]string, []string, error) {
	if len(command.ItemIDs) == 0 || len(command.ItemIDs) > 10 || len(command.CandidateDates) == 0 || len(command.CandidateDates) > 14 {
		return nil, nil, ErrInvalid
	}
	items := unique(command.ItemIDs)
	for index, value := range items {
		normalized, err := normalizeUUID(value)
		if err != nil {
			return nil, nil, err
		}
		items[index] = normalized
	}
	dates := unique(command.CandidateDates)
	for _, value := range dates {
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return nil, nil, ErrInvalid
		}
	}
	sort.Strings(dates)
	return items, dates, nil
}

func requirePatient(principal authn.Principal) error {
	if principal.AccountID == "" || principal.Status != authn.AccountStatusActive {
		return ErrForbidden
	}
	if principal.AccountType != authn.AccountTypePatient && principal.AccountType != authn.AccountTypeStaff {
		return ErrForbidden
	}
	if _, err := uuid.Parse(principal.AccountID); err != nil {
		return ErrForbidden
	}
	return nil
}

func normalizeUUID(value string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", ErrInvalid
	}
	return parsed.String(), nil
}

func unique(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if _, ok := seen[value]; !ok && value != "" {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}

func fingerprint(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
