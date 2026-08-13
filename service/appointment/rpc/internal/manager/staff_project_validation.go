package manager

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	maxCatalogNameRunes        = 128
	maxCatalogDescriptionRunes = 8192
	maxRequestIDBytes          = 64
	defaultProjectPage         = 1
	defaultProjectPageSize     = 20
	maxProjectPageSize         = 100
)

// normalizedCreateCommand validates and canonicalizes every caller-controlled
// field used by Create. Generated state such as item ID, status, version, and
// timestamps is intentionally not part of the command.
func normalizedCreateProjectCommand(command CreateProjectCommand) (CreateProjectCommand, error) {
	var err error
	command.OwnerDepartmentID, err = normalizedUUID(command.OwnerDepartmentID, "owner_department_id")
	if err != nil {
		return CreateProjectCommand{}, err
	}
	command.Name, err = normalizedCatalogName(command.Name)
	if err != nil {
		return CreateProjectCommand{}, err
	}
	command.Description, err = normalizedCatalogDescription(command.Description)
	if err != nil {
		return CreateProjectCommand{}, err
	}
	command.OperationID, command.RequestID, err = normalizedOperation(command.OperationID, command.RequestID)
	if err != nil {
		return CreateProjectCommand{}, err
	}
	return command, nil
}

// normalizedUpdateCommand preserves nil optional fields. A non-nil pointer
// means that the caller intends to replace that field, including validation of
// the replacement value.
func normalizedUpdateProjectCommand(command UpdateProjectCommand) (UpdateProjectCommand, error) {
	var err error
	command.ItemID, err = normalizedUUID(command.ItemID, "item_id")
	if err != nil {
		return UpdateProjectCommand{}, err
	}
	if command.ExpectedVersion <= 0 {
		return UpdateProjectCommand{}, fmt.Errorf("%w: expected_version must be positive", ErrInvalid)
	}
	if command.Name == nil && command.Description == nil {
		return UpdateProjectCommand{}, fmt.Errorf("%w: name or description is required", ErrInvalid)
	}
	if command.Name != nil {
		value, valueErr := normalizedCatalogName(*command.Name)
		if valueErr != nil {
			return UpdateProjectCommand{}, valueErr
		}
		command.Name = &value
	}
	if command.Description != nil {
		value, valueErr := normalizedCatalogDescription(*command.Description)
		if valueErr != nil {
			return UpdateProjectCommand{}, valueErr
		}
		command.Description = &value
	}
	command.OperationID, command.RequestID, err = normalizedOperation(command.OperationID, command.RequestID)
	if err != nil {
		return UpdateProjectCommand{}, err
	}
	return command, nil
}

func normalizedChangeProjectStatusCommand(command ChangeProjectStatusCommand) (ChangeProjectStatusCommand, error) {
	var err error
	command.ItemID, err = normalizedUUID(command.ItemID, "item_id")
	if err != nil {
		return ChangeProjectStatusCommand{}, err
	}
	if command.ExpectedVersion <= 0 {
		return ChangeProjectStatusCommand{}, fmt.Errorf("%w: expected_version must be positive", ErrInvalid)
	}
	command.OperationID, command.RequestID, err = normalizedOperation(command.OperationID, command.RequestID)
	if err != nil {
		return ChangeProjectStatusCommand{}, err
	}
	return command, nil
}

func normalizedListProjectsQuery(query ListProjectsQuery) (ListProjectsQuery, error) {
	var err error
	if strings.TrimSpace(query.OwnerDepartmentID) != "" {
		query.OwnerDepartmentID, err = normalizedUUID(query.OwnerDepartmentID, "owner_department_id")
		if err != nil {
			return ListProjectsQuery{}, err
		}
	}
	query.Status, err = normalizedOptionalStatus(query.Status)
	if err != nil {
		return ListProjectsQuery{}, err
	}
	query.Page, query.PageSize, err = normalizedPage(query.Page, query.PageSize)
	if err != nil {
		return ListProjectsQuery{}, err
	}
	query.RequestID, err = normalizedRequestID(query.RequestID)
	if err != nil {
		return ListProjectsQuery{}, err
	}
	return query, nil
}

func normalizedUUID(value, field string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("%w: %s must be a UUID", ErrInvalid, field)
	}
	return parsed.String(), nil
}

func normalizedOperation(operationID, requestID string) (string, string, error) {
	operationID, err := normalizedUUID(operationID, "operation_id")
	if err != nil {
		return "", "", err
	}
	requestID, err = normalizedRequestID(requestID)
	if err != nil {
		return "", "", err
	}
	return operationID, requestID, nil
}

func normalizedRequestID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) > maxRequestIDBytes {
		return "", fmt.Errorf("%w: request_id exceeds %d bytes", ErrInvalid, maxRequestIDBytes)
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("%w: request_id contains control characters", ErrInvalid)
	}
	return value, nil
}

func normalizedCatalogName(value string) (string, error) {
	return normalizedSingleLineText(value, "name", maxCatalogNameRunes, false)
}

// normalizedCatalogDescription keeps meaningful line breaks and tabs because
// the first project version stores patient-facing preparation rules as natural
// language. Other control characters are rejected.
func normalizedCatalogDescription(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: description is required", ErrInvalid)
	}
	if utf8.RuneCountInString(value) > maxCatalogDescriptionRunes {
		return "", fmt.Errorf("%w: description exceeds %d characters", ErrInvalid, maxCatalogDescriptionRunes)
	}
	if strings.IndexFunc(value, unsupportedDescriptionControl) >= 0 {
		return "", fmt.Errorf("%w: description contains unsupported control characters", ErrInvalid)
	}
	return value, nil
}

func unsupportedDescriptionControl(r rune) bool {
	return unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t'
}

func normalizedSingleLineText(value, field string, maxRunes int, allowEmpty bool) (string, error) {
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

func normalizedOptionalStatus(value Status) (Status, error) {
	value = Status(strings.TrimSpace(string(value)))
	if value == "" {
		return "", nil
	}
	if !value.Valid() {
		return "", fmt.Errorf("%w: unsupported status %q", ErrInvalid, value)
	}
	return value, nil
}

func normalizedPage(page, pageSize int64) (int64, int64, error) {
	if page == 0 {
		page = defaultProjectPage
	}
	if pageSize == 0 {
		pageSize = defaultProjectPageSize
	}
	if page < 1 || pageSize < 1 || pageSize > maxProjectPageSize {
		return 0, 0, fmt.Errorf(
			"%w: page must be positive and page_size must be between 1 and %d",
			ErrInvalid,
			maxProjectPageSize,
		)
	}
	return page, pageSize, nil
}

func pageOffset(page, pageSize int64) (int64, error) {
	const maxInt64 = int64(1<<63 - 1)
	if page < 1 || pageSize < 1 || page-1 > maxInt64/pageSize {
		return 0, fmt.Errorf("%w: page offset exceeds supported range", ErrInvalid)
	}
	return (page - 1) * pageSize, nil
}
