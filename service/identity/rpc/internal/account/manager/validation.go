package manager

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	maxPageSize         = 100
	maxNicknameRunes    = 64
	maxDisplayNameRunes = 128
	maxStaffNoRunes     = 64
	maxDescriptionRunes = 512
	maxAvatarURLBytes   = 2048
)

var mainlandPhone = regexp.MustCompile(`^1[3-9][0-9]{9}$`)

func normalizedPage(page, pageSize int64) (int64, int64, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	if page < 1 || pageSize < 1 || pageSize > maxPageSize {
		return 0, 0, fmt.Errorf("%w: page must be positive and page_size must be between 1 and %d", ErrInvalid, maxPageSize)
	}
	return page, pageSize, nil
}

func normalizedDoctorProfile(profile DoctorProfileInput) (DoctorProfileInput, error) {
	var err error
	profile.DisplayName, err = normalizedText(profile.DisplayName, "display_name", maxDisplayNameRunes, false)
	if err != nil {
		return DoctorProfileInput{}, err
	}
	profile.StaffNo, err = normalizedText(profile.StaffNo, "staff_no", maxStaffNoRunes, true)
	if err != nil {
		return DoctorProfileInput{}, err
	}
	profile.Description, err = normalizedText(profile.Description, "description", maxDescriptionRunes, true)
	if err != nil {
		return DoctorProfileInput{}, err
	}
	profile.AvatarURL, err = normalizedAvatarURL(profile.AvatarURL)
	if err != nil {
		return DoctorProfileInput{}, err
	}
	return profile, nil
}

func normalizedOptionalDoctorProfile(profile OptionalDoctorProfileInput) (OptionalDoctorProfileInput, error) {
	var err error
	if profile.DisplayName != nil {
		value, valueErr := normalizedText(*profile.DisplayName, "display_name", maxDisplayNameRunes, false)
		if valueErr != nil {
			return OptionalDoctorProfileInput{}, valueErr
		}
		profile.DisplayName = &value
	}
	if profile.StaffNo != nil {
		value, valueErr := normalizedText(*profile.StaffNo, "staff_no", maxStaffNoRunes, true)
		if valueErr != nil {
			return OptionalDoctorProfileInput{}, valueErr
		}
		profile.StaffNo = &value
	}
	if profile.Description != nil {
		value, valueErr := normalizedText(*profile.Description, "description", maxDescriptionRunes, true)
		if valueErr != nil {
			return OptionalDoctorProfileInput{}, valueErr
		}
		profile.Description = &value
	}
	if profile.AvatarURL != nil {
		value, valueErr := normalizedAvatarURL(*profile.AvatarURL)
		if valueErr != nil {
			return OptionalDoctorProfileInput{}, valueErr
		}
		profile.AvatarURL = &value
	}
	return profile, err
}

func normalizedAvatarURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if len(value) > maxAvatarURLBytes {
		return "", fmt.Errorf("%w: avatar_url exceeds %d bytes", ErrInvalid, maxAvatarURLBytes)
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("%w: avatar_url must be an HTTP(S) URL", ErrInvalid)
	}
	return value, nil
}

func normalizedText(value, field string, maxRunes int, allowEmpty bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" && !allowEmpty {
		return "", fmt.Errorf("%w: %s is required", ErrInvalid, field)
	}
	if utf8.RuneCountInString(value) > maxRunes {
		return "", fmt.Errorf("%w: %s exceeds %d characters", ErrInvalid, field, maxRunes)
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("%w: %s contains control characters", ErrInvalid, field)
	}
	return value, nil
}

func normalizedUUID(value, field string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("%w: %s must be a UUID", ErrInvalid, field)
	}
	return parsed.String(), nil
}

func validUUID(value, field string) error {
	_, err := normalizedUUID(value, field)
	return err
}

func normalizePhone(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer(" ", "", "-", "").Replace(value)
	value = strings.TrimPrefix(value, "+86")
	if !mainlandPhone.MatchString(value) {
		return "", fmt.Errorf("%w: invalid phone number", ErrInvalid)
	}
	return "+86" + value, nil
}
