package handler

import (
	"github.com/berduk-dev/bad-da-yo/internal/model"
	"github.com/berduk-dev/bad-da-yo/internal/service"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

type Handler struct {
	service service.Service
}

func New(service service.Service) Handler {
	return Handler{
		service: service,
	}
}

func (h *Handler) CreatePrize(c *gin.Context) {
	var req model.PrizeRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, "У вас невалидный запрос")
		log.Println("error PrizeCode ShouldBindJSON:", err)
		return
	}

	code, err := h.service.CreatePrize(c, req.Prize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Произошла ошибка! Попробуйте позже")
		log.Println("error h.service.CreatePrize:", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": code,
	})
}
