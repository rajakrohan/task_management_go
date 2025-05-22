# Task Management System

A microservice-based Task Management System built with Go, PostgreSQL, and Redis.

## Features

- CRUD operations for tasks
- Pagination support
- Status-based filtering
- Redis caching
- PostgreSQL persistence
- RESTful API design

## Prerequisites

- Go 1.22 or later
- PostgreSQL 15 or later
- Redis 7 or later

## Project Structure

```
task-management-service/
├── cmd/
│   └── api/                    # Application entry point
├── internal/
│   ├── domain/                 # Domain models and interfaces
│   ├── repository/             # Data access layer
│   ├── service/                # Business logic
│   └── api/                    # HTTP handlers and middleware
├── pkg/                        # Shared packages
└── config/                     # Configuration management
```

## Database Schema

The system uses PostgreSQL for persistent storage with the following schema:

### Tasks Table
```sql
CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

The `status` column accepts one of the following values:
- `Pending`
- `In Progress`
- `Completed`

## Redis Caching

The system uses Redis for caching to improve performance:

1. **Cache Strategy**:
   - Tasks are cached for 24 hours after creation/update
   - Cache keys follow the pattern: `task:{task_id}`
   - Cache invalidation occurs on task updates and deletions

2. **Cache Operations**:
   - On task creation: Cache the new task
   - On task update: Update the cache
   - On task deletion: Remove from cache
   - On task retrieval: Check cache first, then database

3. **Benefits**:
   - Reduced database load
   - Faster response times for frequently accessed tasks
   - Automatic cache expiration after 24 hours

## Setup and Installation

1. Install Go from https://golang.org/dl/
2. Install PostgreSQL from https://www.postgresql.org/download/
3. Install Redis from https://redis.io/download

### Environment Variables

Create a `.env` file in the root directory:

```env
DATABASE_URL=postgres://postgres:root123@localhost:5432/user
REDIS_HOST=localhost
REDIS_PORT=6379
```

### Database Setup

1. Create a PostgreSQL database:
```sql
CREATE DATABASE user;
```

2. The application will automatically create the required tables on startup.

## Running the Application

1. Clone the repository
2. Install dependencies:
```bash
go mod download
```

3. Run the application:
```bash
go run cmd/api/main.go
```

## API Documentation

### Endpoints

#### Create Task
- **POST** `/tasks`
- Request Body:
```json
{
    "title": "Task Title",
    "description": "Task Description",
    "status": "Pending"  // Required: "Pending", "In Progress", or "Completed"
}
```

#### List Tasks
- **GET** `/tasks`
- Query Parameters:
  - `page`: Page number (default: 1)
  - `size`: Items per page (default: 10)
  - `status`: Filter by status (optional)

#### Get Task by ID
- **GET** `/tasks/{id}`

#### Update Task
- **PUT** `/tasks/{id}`
- Request Body:
```json
{
    "title": "Updated Title",
    "description": "Updated Description",
    "status": "In Progress"
}
```

#### Delete Task
- **DELETE** `/tasks/{id}`

#### Filter by status, combine with pagination:
GET /tasks?status=Completed
GET /tasks?status=Pending
GET /tasks?status=In Progress
GET /tasks?status=Completed&page=1&size=10


## Microservices Architecture

### Design Decisions

1. **Single Responsibility Principle**
   - Each component has a single responsibility
   - Clear separation between domain, repository, service, and API layers

2. **Scalability**
   - Horizontal scaling through stateless design
   - Redis caching for improved performance
   - Database connection pooling

3. **Inter-Service Communication**
   - RESTful APIs for service communication
   - Future services (e.g., User Service) can communicate via:
     - REST APIs for synchronous communication
     - Message queues (e.g., RabbitMQ) for asynchronous operations
     - gRPC for high-performance RPC calls

### Horizontal Scaling

The service can be scaled horizontally by:
1. Running multiple instances behind a load balancer
2. Using Redis for distributed caching
3. Implementing database connection pooling
4. Using environment variables for configuration

## Error Handling

The service implements consistent error handling with appropriate HTTP status codes and error messages.

## Testing

Run tests using:
```bash
go test ./...
```