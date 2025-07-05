package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func PrometheusHandler() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}
