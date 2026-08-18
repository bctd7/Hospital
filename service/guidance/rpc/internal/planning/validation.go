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

func normalizeGenerate(command GenerateCommand) ([]string, []CandidateAvailability, error) {
	if len(command.ItemIDs) == 0 || len(command.ItemIDs) > 10 {
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
	availability := append([]CandidateAvailability(nil), command.CandidateAvailability...)
	if len(availability) == 0 {
		for _, value := range unique(command.CandidateDates) {
			availability = append(availability, CandidateAvailability{ServiceDate: value, Sessions: []string{"morning", "afternoon"}})
		}
	}
	if len(availability) == 0 || len(availability) > 14 {
		return nil, nil, ErrInvalid
	}
	byDate := make(map[string]map[string]bool, len(availability))
	for _, value := range availability {
		date := strings.TrimSpace(value.ServiceDate)
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return nil, nil, ErrInvalid
		}
		if len(value.Sessions) == 0 {
			return nil, nil, ErrInvalid
		}
		if byDate[date] == nil {
			byDate[date] = map[string]bool{}
		}
		for _, session := range unique(value.Sessions) {
			if session != "morning" && session != "afternoon" {
				return nil, nil, ErrInvalid
			}
			byDate[date][session] = true
		}
	}
	dates := make([]string, 0, len(byDate))
	for date := range byDate {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	normalized := make([]CandidateAvailability, 0, len(dates))
	for _, date := range dates {
		sessions := make([]string, 0, 2)
		for _, session := range []string{"morning", "afternoon"} {
			if byDate[date][session] {
				sessions = append(sessions, session)
			}
		}
		normalized = append(normalized, CandidateAvailability{ServiceDate: date, Sessions: sessions})
	}
	return items, normalized, nil
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
