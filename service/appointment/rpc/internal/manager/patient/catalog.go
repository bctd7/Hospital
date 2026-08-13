package patient

import (
	"context"
	"fmt"

	"hospital/common/authn"
)

func (m *Manager) ListProjects(ctx context.Context, patient authn.Principal, departmentID string, page, pageSize int64) (Page[ExaminationItem], error) {
	if err := requirePatient(patient); err != nil {
		return Page[ExaminationItem]{}, err
	}
	departmentID, err := normalizeUUID(departmentID, "department_id")
	if err != nil {
		return Page[ExaminationItem]{}, err
	}
	page, pageSize, offset, err := normalizeBookingPage(page, pageSize)
	if err != nil {
		return Page[ExaminationItem]{}, err
	}
	key := fmt.Sprintf("department:%s:g:%s:patient-projects:%d:%d", departmentID, m.generation(ctx, departmentID), page, pageSize)
	result, _, err := loadCached(ctx, m.cache, &m.flights, key, hotReadCacheTTL, func() (Page[ExaminationItem], bool, error) {
		values, total, loadErr := m.projects.ListItems(ctx, ProjectListFilter{
			OwnerDepartmentID: departmentID, Status: StatusActive, Offset: offset, Limit: pageSize,
		})
		return Page[ExaminationItem]{Items: values, Page: page, PageSize: pageSize, Total: total}, loadErr == nil, loadErr
	})
	return result, err
}
