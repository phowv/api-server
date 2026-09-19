package healthcheck

import (
	"log/slog"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/logger/sl"
	"photo-viewer-server/internal/service"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type healthcheckResponse struct {
	response.Response
	Healthy bool `json:"healthy"`
}

func Healthcheck(lg *slog.Logger, healthCheckService *service.HealthcheckService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := lg.With(
			slog.String("op", "handlers.Healthcheck"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		err := healthCheckService.Check(r.Context())
		if err != nil {
			log.Error("healthcheck isn't pass", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("Unhealthy"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, healthcheckResponse{
			Response: response.OK(),
			Healthy: true,
		})
	}
}
