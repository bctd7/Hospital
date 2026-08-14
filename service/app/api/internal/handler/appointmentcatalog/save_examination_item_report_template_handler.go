package appointmentcatalog

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"hospital/service/app/api/internal/logic/appointmentcatalog"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

func SaveExaminationItemReportTemplateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SaveExaminationItemReportTemplateRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		logic := appointmentcatalog.NewSaveExaminationItemReportTemplateLogic(r.Context(), svcCtx)
		response, err := logic.SaveExaminationItemReportTemplate(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, response)
	}
}
