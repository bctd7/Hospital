// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package organizationdirectory

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"hospital/service/app/api/internal/logic/organizationdirectory"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

func ListDepartmentsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListDepartmentsRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := organizationdirectory.NewListDepartmentsLogic(r.Context(), svcCtx)
		resp, err := l.ListDepartments(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
