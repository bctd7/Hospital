package manager

import (
	"fmt"
	"hospital/service/identity/rpc/internal/organization"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	maxUnitNameRunes  = 128
	maxRequestIDBytes = 64
)

func normalizedUnitName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: name is required", organization.ErrInvalid)
	}
	if utf8.RuneCountInString(value) > maxUnitNameRunes {
		return "", fmt.Errorf("%w: name exceeds %d characters", organization.ErrInvalid, maxUnitNameRunes)
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("%w: name contains control characters", organization.ErrInvalid)
	}
	return value, nil
}

func normalizedUUID(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := uuid.Parse(value)
	if err != nil {
		return "", fmt.Errorf("%w: %s must be a UUID", organization.ErrInvalid, field)
	}
	return parsed.String(), nil
}

func normalizedOperation(operationID, requestID string) (string, string, error) {
	operationID, err := normalizedUUID(operationID, "operation_id")
	if err != nil {
		return "", "", err
	}
	requestID = strings.TrimSpace(requestID)
	if len(requestID) > maxRequestIDBytes {
		return "", "", fmt.Errorf("%w: request_id exceeds %d bytes", organization.ErrInvalid, maxRequestIDBytes)
	}
	return operationID, requestID, nil
}

func generatedUnitCode(unitType organization.UnitType, operationID string) string {
	prefix := "CAMPUS"
	if unitType == organization.UnitTypeDepartment {
		prefix = "DEPT"
	}
	compactID := strings.ReplaceAll(operationID, "-", "")
	return prefix + "-" + strings.ToUpper(compactID[:12])
}

func operationConflict() error {
	return fmt.Errorf("%w: operation_id was already used for a different organization change", organization.ErrConflict)
}
