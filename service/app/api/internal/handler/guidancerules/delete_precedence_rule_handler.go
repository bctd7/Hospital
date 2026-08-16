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

func DeletePrecedenceRuleHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeletePrecedenceRuleRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := guidancerules.NewDeletePrecedenceRuleLogic(r.Context(), svcCtx)
		resp, err := l.DeletePrecedenceRule(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
