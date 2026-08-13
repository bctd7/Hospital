package patient

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager"
)

func normalizeUUID(value, field string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("%w: %s must be a UUID", appointmentmanager.ErrInvalid, field)
	}
	return parsed.String(), nil
}
