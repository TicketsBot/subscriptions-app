package server

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (s *Server) ErrorHandler(ctx *gin.Context) {
	ctx.Next()

	for _, err := range ctx.Errors {
		s.logger.Error(
			err.Error(),
			zap.Any("meta", err.Meta),
			zap.Any("stack", err.Err),
			zap.String("path", ctx.Request.URL.Path),
			zap.String("method", ctx.Request.Method),
			zap.Int("status", ctx.Writer.Status()),
		)
	}

	if len(ctx.Errors) > 0 {
		ctx.JSON(500, errorJson("Internal server error"))
	}
}
