package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"task-management-service/internal/domain"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// TaskRepository defines the interface for task data access
type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	List(ctx context.Context, page, size int, status domain.TaskStatus) ([]*domain.Task, int64, error)
	Update(ctx context.Context, task *domain.Task) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// PostgresTaskRepository implements TaskRepository using PostgreSQL
type PostgresTaskRepository struct {
	db    *sql.DB
	redis *redis.Client
}

// NewPostgresTaskRepository creates a new PostgreSQL task repository
func NewPostgresTaskRepository(db *sql.DB, redis *redis.Client) *PostgresTaskRepository {
	return &PostgresTaskRepository{
		db:    db,
		redis: redis,
	}
}

// Create implements TaskRepository
func (r *PostgresTaskRepository) Create(ctx context.Context, task *domain.Task) error {
	query := `
		INSERT INTO tasks (id, title, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		task.ID, task.Title, task.Description, task.Status,
		task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Cache the task
	if err := r.cacheTask(ctx, task); err != nil {
		// Log the error but don't fail the operation
		fmt.Printf("failed to cache task: %v\n", err)
	}

	return nil
}

// GetByID implements TaskRepository
func (r *PostgresTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	// Try to get from cache first
	task, err := r.getTaskFromCache(ctx, id)
	if err == nil && task != nil {
		return task, nil
	}

	query := `
		SELECT id, title, description, status, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`
	task = &domain.Task{}
	err = r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &task.Title, &task.Description, &task.Status,
		&task.CreatedAt, &task.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Cache the task
	if err := r.cacheTask(ctx, task); err != nil {
		fmt.Printf("failed to cache task: %v\n", err)
	}

	return task, nil
}

// List implements TaskRepository
func (r *PostgresTaskRepository) List(ctx context.Context, page, size int, status domain.TaskStatus) ([]*domain.Task, int64, error) {
	offset := (page - 1) * size

	var query string
	var args []interface{}
	var countQuery string
	var countArgs []interface{}

	if status != "" {
		query = `
			SELECT id, title, description, status, created_at, updated_at
			FROM tasks
			WHERE status = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		countQuery = `SELECT COUNT(*) FROM tasks WHERE status = $1`
		args = []interface{}{status, size, offset}
		countArgs = []interface{}{status}
	} else {
		query = `
			SELECT id, title, description, status, created_at, updated_at
			FROM tasks
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`
		countQuery = `SELECT COUNT(*) FROM tasks`
		args = []interface{}{size, offset}
		countArgs = []interface{}{}
	}

	// Get total count
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get task count: %w", err)
	}

	// Get tasks
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		task := &domain.Task{}
		err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &task.Status,
			&task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, total, nil
}

// Update implements TaskRepository
func (r *PostgresTaskRepository) Update(ctx context.Context, task *domain.Task) error {
	query := `
		UPDATE tasks
		SET title = $1, description = $2, status = $3, updated_at = $4
		WHERE id = $5
	`
	result, err := r.db.ExecContext(ctx, query,
		task.Title, task.Description, task.Status,
		task.UpdatedAt, task.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrTaskNotFound
	}

	// Update cache
	if err := r.cacheTask(ctx, task); err != nil {
		fmt.Printf("failed to cache task: %v\n", err)
	}

	return nil
}

// Delete implements TaskRepository
func (r *PostgresTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tasks WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrTaskNotFound
	}

	// Delete from cache
	if err := r.deleteTaskFromCache(ctx, id); err != nil {
		fmt.Printf("failed to delete task from cache: %v\n", err)
	}

	return nil
}

// Helper methods for caching
func (r *PostgresTaskRepository) cacheTask(ctx context.Context, task *domain.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	key := fmt.Sprintf("task:%s", task.ID.String())
	return r.redis.Set(ctx, key, data, 24*time.Hour).Err()
}

func (r *PostgresTaskRepository) getTaskFromCache(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	key := fmt.Sprintf("task:%s", id.String())
	data, err := r.redis.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task from cache: %w", err)
	}

	var task domain.Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

func (r *PostgresTaskRepository) deleteTaskFromCache(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("task:%s", id.String())
	return r.redis.Del(ctx, key).Err()
}
