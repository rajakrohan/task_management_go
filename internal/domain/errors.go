package domain

import "errors"

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrInvalidTaskTitle  = errors.New("task title cannot be empty")
	ErrInvalidTaskStatus = errors.New("invalid task status")
) 