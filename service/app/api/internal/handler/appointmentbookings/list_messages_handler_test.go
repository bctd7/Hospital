package appointmentbookings

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func TestListMessagesRequestAllowsSuperAdminOverviewWithoutDepartment(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/appointment/messages?page=1&page_size=100", nil)
	var input types.ListMessagesAPIRequest
	if err := httpx.Parse(request, &input); err != nil {
		t.Fatalf("parse message overview request: %v", err)
	}
	if input.DepartmentID != "" || input.Page != 1 || input.PageSize != 100 {
		t.Fatalf("unexpected parsed input: %#v", input)
	}
}
