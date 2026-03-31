package ports

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates a Chi router with all planning routes
func NewRouter(h *HTTPHandler) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/v1", func(r chi.Router) {
		r.Post("/plans", h.HandleCreatePlan)
		r.Get("/plans", h.HandleListPlans)
		r.Get("/plans/{planId}", func(w http.ResponseWriter, rq *http.Request) {
			h.HandleGetPlan(w, rq, chi.URLParam(rq, "planId"))
		})
		r.Get("/plans/{planId}/items", func(w http.ResponseWriter, rq *http.Request) {
			h.HandleGetPlanItems(w, rq, chi.URLParam(rq, "planId"))
		})
		r.Post("/plans/{planId}/items", func(w http.ResponseWriter, rq *http.Request) {
			h.HandleAddItem(w, rq, chi.URLParam(rq, "planId"))
		})
		r.Delete("/plans/{planId}/items/{itemId}", func(w http.ResponseWriter, rq *http.Request) {
			h.HandleRemoveItem(w, rq, chi.URLParam(rq, "planId"), chi.URLParam(rq, "itemId"))
		})
		r.Post("/plans/{planId}/process", func(w http.ResponseWriter, rq *http.Request) {
			h.HandleProcessPlan(w, rq, chi.URLParam(rq, "planId"))
		})
		r.Post("/plans/{planId}/hold", func(w http.ResponseWriter, rq *http.Request) {
			h.HandleHoldPlan(w, rq, chi.URLParam(rq, "planId"))
		})
		r.Post("/plans/{planId}/resume", func(w http.ResponseWriter, rq *http.Request) {
			h.HandleResumePlan(w, rq, chi.URLParam(rq, "planId"))
		})
		r.Post("/plans/{planId}/release", func(w http.ResponseWriter, rq *http.Request) {
			h.HandleReleasePlan(w, rq, chi.URLParam(rq, "planId"))
		})
		r.Post("/plans/{planId}/complete", func(w http.ResponseWriter, rq *http.Request) {
			h.HandleCompletePlan(w, rq, chi.URLParam(rq, "planId"))
		})
		r.Post("/plans/{planId}/cancel", func(w http.ResponseWriter, rq *http.Request) {
			h.HandleCancelPlan(w, rq, chi.URLParam(rq, "planId"))
		})
	})

	return r
}
