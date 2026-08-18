package guidanceconfiguration

import (
	"net/http"

	"hospital/service/app/api/internal/logic/guidanceconfiguration"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetExaminationItemPatientRemindersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ExaminationItemConfigurationPathRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		logic := guidanceconfiguration.NewGetExaminationItemPatientRemindersLogic(r.Context(), svcCtx)
		resp, err := logic.GetExaminationItemPatientReminders(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
