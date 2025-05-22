package service

import (
	"context"
	"fmt"

	"task-management-service/internal/domain"
	"task-management-service/internal/repository"

	"github.com/google/uuid"
)

// TaskService handles the business logic for tasks
type TaskService struct {
	repo repository.TaskRepository
}

// NewTaskService creates a new task service
func NewTaskService(repo repository.TaskRepository) *TaskService {
	return &TaskService{
		repo: repo,
	}
}

// CreateTask creates a new task
func (s *TaskService) CreateTask(ctx context.Context, title, description string, status domain.TaskStatus) (*domain.Task, error) {
	task := domain.NewTask(title, description, status)
	if err := task.Validate(); err != nil {
		return nil, fmt.Errorf("invalid task: %w", err)
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}

// GetTask retrieves a task by ID
func (s *TaskService) GetTask(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return task, nil
}

// ListTasks retrieves a list of tasks with pagination and status filtering
func (s *TaskService) ListTasks(ctx context.Context, page, size int, status domain.TaskStatus) ([]*domain.Task, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}

	tasks, total, err := s.repo.List(ctx, page, size, status)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	return tasks, total, nil
}

// UpdateTask updates an existing task
func (s *TaskService) UpdateTask(ctx context.Context, id uuid.UUID, title, description string, status domain.TaskStatus) (*domain.Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	task.Update(title, description, status)
	if err := task.Validate(); err != nil {
		return nil, fmt.Errorf("invalid task: %w", err)
	}

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return task, nil
}

// DeleteTask deletes a task by ID
func (s *TaskService) DeleteTask(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	return nil
}
