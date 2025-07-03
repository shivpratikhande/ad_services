package handlers

import (
	"net/http"
	"strconv"

	"ad-tracker/internal/models"
	"ad-tracker/internal/repository"

	"github.com/gin-gonic/gin"
)

type AdHandler struct {
	repo *repository.AdRepository
}

func NewAdHandler(repo *repository.AdRepository) *AdHandler {
	return &AdHandler{repo: repo}
}

func (h *AdHandler) CreateAdEvent(c *gin.Context) {
	var req models.AdEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	event, err := h.repo.CreateAdEvent(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, event)
}

func (h *AdHandler) GetAdEvents(c *gin.Context) {
	campaignID := c.Param("campaignId")

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter"})
		return
	}

	events, err := h.repo.GetAdEvents(campaignID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"meta": gin.H{
			"limit":  limit,
			"offset": offset,
			"count":  len(events),
		},
	})
}

func (h *AdHandler) GetCampaignSummary(c *gin.Context) {
	campaignID := c.Param("campaignId")

	summary, err := h.repo.GetCampaignSummary(campaignID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

func (h *AdHandler) GetAnalytics(c *gin.Context) {
	campaignID := c.Param("campaignId")

	// Parse days parameter
	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid days parameter"})
		return
	}

	analytics, err := h.repo.GetAnalytics(campaignID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"analytics": analytics,
		"meta": gin.H{
			"campaign_id": campaignID,
			"days":        days,
		},
	})
}

func (h *AdHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "ad-tracker",
	})
}
