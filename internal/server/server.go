package server

import (
	"github.com/TicketsBot/subscriptions-app/internal/config"
	"github.com/TicketsBot/subscriptions-app/pkg/patreon"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"sync"
	"time"
)

type Server struct {
	config config.Config
	logger *zap.Logger

	pledges map[string]patreon.Patron
	mu      sync.RWMutex
}

func NewServer(config config.Config, logger *zap.Logger) *Server {
	return &Server{
		config: config,
		logger: logger,
	}
}

func (s *Server) Run() error {
	router := gin.New()

	router.Use(ginzap.Ginzap(s.logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(s.logger, true))
	router.Use(s.ErrorHandler)

	router.POST("/interaction", s.Authenticate, s.HandleInteraction)

	return router.Run(s.config.ServerAddr)
}

func (s *Server) UpdatePledges(pledges map[string]patreon.Patron) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pledges = pledges
}
