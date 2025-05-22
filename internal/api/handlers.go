package api

import (
	"fmt"
	"net/http"
	"strconv"

	"task-management-service/internal/domain"
	"task-management-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TaskHandler handles HTTP requests for tasks
type TaskHandler struct {
	service *service.TaskService
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(service *service.TaskService) *TaskHandler {
	return &TaskHandler{
		service: service,
	}
}

// RegisterRoutes registers the task routes
func (h *TaskHandler) RegisterRoutes(router *gin.Engine) {
	tasks := router.Group("/tasks")
	{
		tasks.POST("", h.CreateTask)
		tasks.GET("", h.ListTasks)
		tasks.GET("/:id", h.GetTask)
		tasks.PUT("/:id", h.UpdateTask)
		tasks.DELETE("/:id", h.DeleteTask)
	}
}

// CreateTaskRequest represents the request body for creating a task
type CreateTaskRequest struct {
	Title       string            `json:"title" binding:"required"`
	Description string            `json:"description"`
	Status      domain.TaskStatus `json:"status" binding:"required,oneof=Pending 'In Progress' Completed"`
}

// UpdateTaskRequest represents the request body for updating a task
type UpdateTaskRequest struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      domain.TaskStatus `json:"status"`
}

// CreateTask handles the creation of a new task
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.service.CreateTask(c.Request.Context(), req.Title, req.Description, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// ListTasks handles the retrieval of tasks with pagination and filtering
func (h *TaskHandler) ListTasks(c *gin.Context) {
	// Parse and validate pagination parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	size, err := strconv.Atoi(c.DefaultQuery("size", "10"))
	if err != nil || size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}

	// Parse and validate status filter
	statusStr := c.Query("status")
	var status domain.TaskStatus
	if statusStr != "" {
		status = domain.TaskStatus(statusStr)
		// Validate status
		switch status {
		case domain.StatusPending, domain.StatusInProgress, domain.StatusCompleted:
			// Valid status
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("invalid status filter. Must be one of: %s, %s, %s",
					domain.StatusPending, domain.StatusInProgress, domain.StatusCompleted),
			})
			return
		}
	}

	tasks, total, err := h.service.ListTasks(c.Request.Context(), page, size, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate pagination metadata
	totalPages := (total + int64(size) - 1) / int64(size)
	hasNext := int64(page) < totalPages
	hasPrev := page > 1

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"pagination": gin.H{
			"total":       total,
			"page":        page,
			"size":        size,
			"total_pages": totalPages,
			"has_next":    hasNext,
			"has_prev":    hasPrev,
		},
		"filter": gin.H{
			"status": statusStr,
		},
	})
}

// GetTask handles the retrieval of a single task
func (h *TaskHandler) GetTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	task, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

// UpdateTask handles the update of an existing task
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.service.UpdateTask(c.Request.Context(), id, req.Title, req.Description, req.Status)
	if err != nil {
		if err == domain.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

// DeleteTask handles the deletion of a task
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	if err := h.service.DeleteTask(c.Request.Context(), id); err != nil {
		if err == domain.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
