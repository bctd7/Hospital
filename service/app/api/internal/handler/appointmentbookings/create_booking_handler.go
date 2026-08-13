// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentbookings

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"hospital/service/app/api/internal/logic/appointmentbookings"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

func CreateBookingHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateBookingAPIRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := appointmentbookings.NewCreateBookingLogic(r.Context(), svcCtx)
		resp, err := l.CreateBooking(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
