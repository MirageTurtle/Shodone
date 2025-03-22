package api

import (
	"context"
	// "encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"shodone/internal/client"
	"shodone/internal/config"
	"shodone/internal/storage"
)

// Server represents the API server
type Server struct {
	router   *gin.Engine
	client   *client.Client
	db       *storage.DB
	cfg      *config.Config
	logger   *log.Logger
	server   *http.Server
	keyMutex sync.Mutex
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, db *storage.DB, logger *log.Logger) *Server {
	// Create API client
	apiClient := client.New(cfg.APIHost)

	// Create server
	server := &Server{
		router:   gin.Default(),
		client:   apiClient,
		db:       db,
		cfg:      cfg,
		logger:   logger,
		keyMutex: sync.Mutex{},
	}

	// Setup routes
	server.setupRoutes()

	return server
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Config endpoints
	configGroup := s.router.Group("/config")
	{
		configGroup.GET("/", s.getConfig)
		configGroup.PUT("/api-host", s.setAPIHost)
	}

	// API key management
	keyGroup := s.router.Group("/keys")
	{
		keyGroup.GET("/", s.getAllAPIKeys)
		keyGroup.POST("/", s.addAPIKey)
		keyGroup.GET("/:id", s.getAPIKey)
		keyGroup.DELETE("/:id", s.deleteAPIKey)
		keyGroup.PUT("/:id", s.updateAPIKey)
		keyGroup.GET("/refresh", s.refreshAPIKeys)
	}

	// API proxy endpoint - match any path under /api
	s.router.Any("/api/*path", s.proxyRequest)
}

// Start starts the API server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	s.logger.Printf("Server listening on %s", addr)
	return s.server.ListenAndServe()
}

// Stop stops the API server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// getConfig returns the current configuration
func (s *Server) getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"api_host":            s.cfg.APIHost,
		"port":                s.cfg.Port,
		"default_quota_limit": s.cfg.DefaultQuotaLimit,
		"cost_per_request":    s.cfg.CostPerRequest,
	})
}

// setAPIHost sets the API host
func (s *Server) setAPIHost(c *gin.Context) {
	var req struct {
		APIHost string `json:"api_host" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update config
	s.cfg.APIHost = req.APIHost
	s.client.SetBaseURL(req.APIHost)

	// Save config
	if err := s.cfg.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save configuration"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "api_host": req.APIHost})
}

// getAllAPIKeys returns all API keys
func (s *Server) getAllAPIKeys(c *gin.Context) {
	keys, err := s.db.GetAllAPIKeys()
	if err != nil {
		s.logger.Printf("Failed to get API keys: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get API keys"})
		return
	}

	// Mask the actual key values for security
	for _, key := range keys {
		key.Key = maskAPIKey(key.Key)
	}

	c.JSON(http.StatusOK, keys)
}

// getAPIKey returns a specific API key
func (s *Server) getAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid key ID"})
		return
	}

	key, err := s.db.GetAPIKey(id)
	if err != nil {
		s.logger.Printf("Failed to get API key %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get API key"})
		return
	}

	// Mask the actual key value for security
	key.Key = maskAPIKey(key.Key)

	c.JSON(http.StatusOK, key)
}

// addAPIKey adds a new API key
func (s *Server) addAPIKey(c *gin.Context) {
	var req struct {
		Key         string    `json:"key" binding:"required"`
		QuotaLimit  int       `json:"quota_limit"`
		RefreshesAt time.Time `json:"refreshes_at"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If quota limit is not provided, use the default
	if req.QuotaLimit <= 0 {
		req.QuotaLimit = s.cfg.DefaultQuotaLimit
	}

	// If refresh date is not provided, set to one month from now
	if req.RefreshesAt.IsZero() {
		req.RefreshesAt = time.Now().AddDate(0, 1, 0)
	}

	// Add the API key
	id, err := s.db.AddAPIKey(req.Key, req.QuotaLimit, req.RefreshesAt)
	if err != nil {
		s.logger.Printf("Failed to add API key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add API key"})
		return
	}

	key, err := s.db.GetAPIKey(id)
	if err != nil {
		s.logger.Printf("Failed to get added API key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve added API key"})
		return
	}

	// Mask the actual key value for security
	key.Key = maskAPIKey(key.Key)

	c.JSON(http.StatusCreated, key)
}

// deleteAPIKey deletes an API key
func (s *Server) deleteAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid key ID"})
		return
	}

	if err := s.db.DeleteAPIKey(id); err != nil {
		s.logger.Printf("Failed to delete API key %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// updateAPIKey updates an API key
func (s *Server) updateAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid key ID"})
		return
	}

	var req struct {
		IsActive   *bool  `json:"is_active"`
		QuotaLimit *int   `json:"quota_limit"`
		QuotaUsed  *int   `json:"quota_used"`
		Key        string `json:"key"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current key
	key, err := s.db.GetAPIKey(id)
	if err != nil {
		s.logger.Printf("Failed to get API key %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get API key"})
		return
	}

	// Update fields if provided
	if req.IsActive != nil {
		if err := s.db.UpdateAPIKeyStatus(id, *req.IsActive, key.ErrorCount); err != nil {
			s.logger.Printf("Failed to update API key status: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update API key"})
			return
		}
	}

	// Update other fields as needed
	// This is simplified - you might want to add more comprehensive update logic

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// refreshAPIKeys checks all API keys and updates their status
func (s *Server) refreshAPIKeys(c *gin.Context) {
	keys, err := s.db.GetAllAPIKeys()
	if err != nil {
		s.logger.Printf("Failed to get API keys: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get API keys"})
		return
	}

	var updatedCount int
	for _, key := range keys {
		// Check if key is valid and get remaining quota
		isValid, remainingQuota, err := s.client.CheckAPIKey(key.Key)
		if err != nil {
			s.logger.Printf("Failed to check API key %d: %v", key.ID, err)
			continue
		}
		// Update quota used
		// ...

		// Update key status
		if !isValid && key.IsActive {
			if err := s.db.UpdateAPIKeyStatus(key.ID, false, key.ErrorCount+1); err != nil {
				s.logger.Printf("Failed to update API key status: %v", err)
				continue
			}
			updatedCount++
		} else if isValid && !key.IsActive {
			if err := s.db.UpdateAPIKeyStatus(key.ID, true, 0); err != nil {
				s.logger.Printf("Failed to update API key status: %v", err)
				continue
			}
			updatedCount++
		}

		// Update quota if needed (this would depend on your API behavior)
		// This is just an example - you might need to customize this logic
		if isValid && remainingQuota > 0 && key.QuotaLimit != remainingQuota {
			// In this example, we're assuming the API tells us the total remaining quota
			// You might need to adjust this based on your API's response
			quotaUsed := key.QuotaLimit - remainingQuota
			// Update quota used
			// This is simplified - you might need more complex logic
			if err := s.db.UpdateAPIKeyUsage(key.ID, quotaUsed); err != nil {
				s.logger.Printf("Failed to update API key usage: %v", err)
				continue
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"total_keys":   len(keys),
		"updated_keys": updatedCount,
	})
}

// proxyRequest proxies a request to the configured API
func (s *Server) proxyRequest(c *gin.Context) {
	// Extract path from URL
	path := c.Param("path")

	// Get an available API key
	s.keyMutex.Lock()
	key, err := s.db.GetAvailableAPIKey()
	if err != nil {
		s.keyMutex.Unlock()
		s.logger.Printf("Failed to get available API key: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No available API keys"})
		return
	}

	// Increment usage before making the request
	// This prevents simultaneous requests from exceeding quota
	if err := s.db.UpdateAPIKeyUsage(key.ID, s.cfg.CostPerRequest); err != nil {
		s.keyMutex.Unlock()
		s.logger.Printf("Failed to update API key usage: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update API key usage"})
		return
	}
	s.keyMutex.Unlock()

	// Forward the request to the API
	resp, err := s.client.Do(c.Request.Method, path, c.Request.Body, key.Key)
	if err != nil {
		s.logger.Printf("API request failed: %v", err)

		// If the request failed, try to restore the quota (optional)
		if updateErr := s.db.UpdateAPIKeyUsage(key.ID, -s.cfg.CostPerRequest); updateErr != nil {
			s.logger.Printf("Failed to restore API key usage: %v", updateErr)
		}

		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach API"})
		return
	}
	defer resp.Body.Close()

	// Check if the response indicates an API key error
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		// Update key status
		if err := s.db.UpdateAPIKeyStatus(key.ID, false, key.ErrorCount+1); err != nil {
			s.logger.Printf("Failed to update API key status: %v", err)
		}
	}

	// Copy headers from API response
	for k, v := range resp.Header {
		c.Writer.Header()[k] = v
	}
	c.Writer.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(c.Writer, resp.Body)
}

// maskAPIKey masks the API key for display purposes
func maskAPIKey(key string) string {
	if len(key) < 4 {
		return "****"
	}
	return key[:4] + "****"
}
