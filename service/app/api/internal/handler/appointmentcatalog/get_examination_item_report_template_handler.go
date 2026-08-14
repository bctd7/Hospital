package appointmentcatalog

import (
	"net/http"

	"hospital/service/app/api/internal/logic/appointmentcatalog"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetExaminationItemReportTemplateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ExaminationItemPathRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		logic := appointmentcatalog.NewGetExaminationItemReportTemplateLogic(r.Context(), svcCtx)
		response, err := logic.GetExaminationItemReportTemplate(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, response)
	}
}
