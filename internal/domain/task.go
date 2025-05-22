package domain

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus represents the possible states of a task
type TaskStatus string

const (
	StatusPending    TaskStatus = "Pending"
	StatusInProgress TaskStatus = "In Progress"
	StatusCompleted  TaskStatus = "Completed"
)

// Task represents the core domain model for a task
type Task struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// NewTask creates a new task with default values
func NewTask(title, description string, status TaskStatus) *Task {
	now := time.Now()
	if status == "" {
		status = StatusPending
	}
	return &Task{
		ID:          uuid.New(),
		Title:       title,
		Description: description,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Validate validates the task fields
func (t *Task) Validate() error {
	if t.Title == "" {
		return ErrInvalidTaskTitle
	}
	if t.Status != StatusPending && t.Status != StatusInProgress && t.Status != StatusCompleted {
		return ErrInvalidTaskStatus
	}
	return nil
}

// Update updates the task fields
func (t *Task) Update(title, description string, status TaskStatus) {
	if title != "" {
		t.Title = title
	}
	if description != "" {
		t.Description = description
	}
	if status != "" {
		t.Status = status
	}
	t.UpdatedAt = time.Now()
}
