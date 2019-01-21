package utils

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"hexmeet.com/haishen/tuna/logp"
)

//Ginzap gin log middleware using zap
func Ginzap(logger *logp.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// some evil middlewares modify this values
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		logger.With(zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("ip", c.ClientIP())).Debug(path)

		c.Next()

		end := time.Now()
		latency := int(end.Sub(start) / 1000000)

		if len(c.Errors) > 0 {
			// Append error field if this is an erroneous request.
			for _, e := range c.Errors.Errors() {
				logger.Error(e)
			}
		} else {
			logger.With(zap.Int("status", c.Writer.Status()),
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.String("query", query),
				zap.String("ip", c.ClientIP()),
				zap.String("user-agent", c.Request.UserAgent()),
				zap.Int("latency", latency)).Debug(path)
		}
	}
}
