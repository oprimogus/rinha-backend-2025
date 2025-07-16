package payment

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	externalservices "github.com/oprimogus/rinha-backend-2025/internal/core/external_services"
	"github.com/oprimogus/rinha-backend-2025/internal/infra/database"
	"github.com/oprimogus/rinha-backend-2025/internal/infra/xerror"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) getHealthStatus(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		xerr := xerror.NewCustomError(http.StatusBadRequest, "missing processor name", nil)
		w.WriteHeader(xerr.Code)
		json.NewEncoder(w).Encode(xerr)
		return
	}

	processorName := externalservices.ProcessorName(name)

	switch processorName {
	default:
		xerr := xerror.NewCustomError(http.StatusBadRequest, "invalid processor name", nil)
		w.WriteHeader(xerr.Code)
		json.NewEncoder(w).Encode(xerr)
		return
	case externalservices.ProcessorDefault, externalservices.ProcessorFallback:
		h, err := h.service.GetHealthStatus(r.Context(), processorName)
		if err != nil {
			xerr := xerror.NewCustomError(http.StatusInternalServerError, "internal error", err)
			w.WriteHeader(xerr.Code)
			json.NewEncoder(w).Encode(xerr)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(h)
		return
	}
}

func (h *Handler) getPaymentsSummary(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if (from != "" && to == "") || (from == "" && to != "") {
		xerr := xerror.NewCustomError(http.StatusBadRequest, "both 'from' and 'to' must be provided, or neither", nil)
		w.WriteHeader(xerr.Code)
		json.NewEncoder(w).Encode(xerr)
		return
	}

	params := PaymentSummaryParams{
		Filter: from != "" && to != "",
	}

	if params.Filter {
		tFrom, err := time.Parse(time.RFC3339, from)
		if err != nil {
			xerr := xerror.NewCustomError(http.StatusBadRequest, "invalid 'from' date", err)
			w.WriteHeader(xerr.Code)
			json.NewEncoder(w).Encode(xerr)
			return
		}
		params.From = tFrom

		tTo, err := time.Parse(time.RFC3339Nano, to)
		if err != nil {
			xerr := xerror.NewCustomError(http.StatusBadRequest, "invalid 'to' date", err)
			w.WriteHeader(xerr.Code)
			json.NewEncoder(w).Encode(xerr)
			return
		}
		params.To = tTo
	}

	summary, err := h.service.GetPaymentsSummary(r.Context(), params)
	if err != nil {
		xerr := xerror.NewCustomError(http.StatusInternalServerError, "internal server error", err)
		w.WriteHeader(xerr.Code)
		json.NewEncoder(w).Encode(xerr)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(summary)
}

func (h *Handler) postPayment(w http.ResponseWriter, r *http.Request) {
	var params PaymentParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		xerr := xerror.NewCustomError(http.StatusBadRequest, "invalid payment data", err)
		w.WriteHeader(xerr.Code)
		json.NewEncoder(w).Encode(xerr)
		return
	}
	
	payment, err := h.service.ProcessPayment(r.Context(), params)
	if err != nil {
		xerr := xerror.NewCustomError(http.StatusInternalServerError, "internal server error", err)
		w.WriteHeader(xerr.Code)
		json.NewEncoder(w).Encode(xerr)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(payment)
}

func SetupRoutes(r *chi.Mux, db *database.Redis) {
	repository := NewRepository(db)
	handler := NewHandler(NewService(repository))
	r.Get("/external-services/health", handler.getHealthStatus)
	r.Get("/payments-summary", handler.getPaymentsSummary)
	r.Post("/payments", handler.postPayment)
}
