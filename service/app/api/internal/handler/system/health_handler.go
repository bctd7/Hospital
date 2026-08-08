// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package system

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"hospital/service/app/api/internal/logic/system"
	"hospital/service/app/api/internal/svc"
)

func HealthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := system.NewHealthLogic(r.Context(), svcCtx)
		resp, err := l.Health()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
