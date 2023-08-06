package server

import "github.com/gin-gonic/gin"

func errorJson(message string) gin.H {
	return gin.H{
		"error": message,
	}
}

func contains[T comparable](slice []T, item T) bool {
	for _, i := range slice {
		if i == item {
			return true
		}
	}

	return false
}

func ptr[T any](value T) *T {
	return &value
}
