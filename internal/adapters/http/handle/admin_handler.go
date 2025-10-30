package handle

import (
	"net/http"

	"ride-hail/internal/core/ports"
	"ride-hail/pkg/logger"
)

type AdminHandler struct {
	service ports.AdminService
	log     *logger.Logger
}

func NewAdminHandler(service ports.AdminService, log *logger.Logger) *AdminHandler {
	return &AdminHandler{
		service: service,
		log:     log,
	}
}

// GetSystemOverview обрабатывает запрос GET /admin/overview
func (h *AdminHandler) GetSystemOverview(w http.ResponseWriter, r *http.Request) {
	// Реализация обработчика
}

// GetActiveRides обрабатывает запрос GET /admin/rides/active
func (h *AdminHandler) GetActiveRides(w http.ResponseWriter, r *http.Request) {
	// Реализация обработчика
}
