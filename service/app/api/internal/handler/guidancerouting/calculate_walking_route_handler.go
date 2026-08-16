// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package guidancerouting

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"hospital/service/app/api/internal/logic/guidancerouting"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

func CalculateWalkingRouteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CalculateWalkingRouteRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := guidancerouting.NewCalculateWalkingRouteLogic(r.Context(), svcCtx)
		resp, err := l.CalculateWalkingRoute(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
