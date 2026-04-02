package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"bonsai_server/internal/usecase"
)

type SensorHandler struct {
	sensorService *usecase.SensorService
}

func NewSensorHandler(sensorService *usecase.SensorService) *SensorHandler {
	return &SensorHandler{sensorService: sensorService}
}

func (h *SensorHandler) GetCurrent(c *gin.Context) {
	sensors, err := h.sensorService.Current(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to load sensors"})
		return
	}

	c.JSON(http.StatusOK, sensors)
}
