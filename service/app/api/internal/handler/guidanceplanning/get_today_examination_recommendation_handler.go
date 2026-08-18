// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package guidanceplanning

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"hospital/service/app/api/internal/logic/guidanceplanning"
	"hospital/service/app/api/internal/svc"
)

func GetTodayExaminationRecommendationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := guidanceplanning.NewGetTodayExaminationRecommendationLogic(r.Context(), svcCtx)
		resp, err := l.GetTodayExaminationRecommendation()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
