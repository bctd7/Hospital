// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package identityadmin

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"hospital/service/app/api/internal/logic/identityadmin"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

func EnableManagedAccountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ManagedAccountMutationRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := identityadmin.NewEnableManagedAccountLogic(r.Context(), svcCtx)
		resp, err := l.EnableManagedAccount(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
