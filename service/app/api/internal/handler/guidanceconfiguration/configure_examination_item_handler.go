// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package guidanceconfiguration

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"hospital/service/app/api/internal/logic/guidanceconfiguration"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

func ConfigureExaminationItemHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ConfigureExaminationItemRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := guidanceconfiguration.NewConfigureExaminationItemLogic(r.Context(), svcCtx)
		resp, err := l.ConfigureExaminationItem(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
