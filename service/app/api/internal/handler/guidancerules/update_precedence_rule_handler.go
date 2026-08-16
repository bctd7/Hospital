// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package guidancerules

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"hospital/service/app/api/internal/logic/guidancerules"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

func UpdatePrecedenceRuleHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdatePrecedenceRuleRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := guidancerules.NewUpdatePrecedenceRuleLogic(r.Context(), svcCtx)
		resp, err := l.UpdatePrecedenceRule(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
