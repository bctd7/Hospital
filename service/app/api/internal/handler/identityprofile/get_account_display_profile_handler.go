// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package identityprofile

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"hospital/service/app/api/internal/logic/identityprofile"
	"hospital/service/app/api/internal/svc"
)

func GetAccountDisplayProfileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := identityprofile.NewGetAccountDisplayProfileLogic(r.Context(), svcCtx)
		resp, err := l.GetAccountDisplayProfile()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
