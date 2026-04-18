package recurringtask

import (
	"context"
	"time"

	recurringtaskdomain "example.com/taskservice/internal/domain/recurringtask"
	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository interface {
	Create(ctx context.Context, template *recurringtaskdomain.Template) (*recurringtaskdomain.Template, error)
	GetByID(ctx context.Context, id int64) (*recurringtaskdomain.Template, error)
	Update(ctx context.Context, template *recurringtaskdomain.Template) (*recurringtaskdomain.Template, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]recurringtaskdomain.Template, error)
}

type TaskRepository interface {
	CreateGeneratedFromTemplate(ctx context.Context, template *recurringtaskdomain.Template, scheduledFor time.Time, createdAt time.Time) (*taskdomain.Task, error)
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*recurringtaskdomain.Template, error)
	GetByID(ctx context.Context, id int64) (*recurringtaskdomain.Template, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*recurringtaskdomain.Template, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]recurringtaskdomain.Template, error)
	Generate(ctx context.Context, input GenerateInput) ([]taskdomain.Task, error)
}

type CreateInput struct {
	Title       string
	Description string
	Active      bool
	Recurrence  RecurrenceInput
}

type UpdateInput struct {
	Title       string
	Description string
	Active      bool
	Recurrence  RecurrenceInput
}

type RecurrenceInput struct {
	Type         recurringtaskdomain.Type
	StartDate    time.Time
	EndDate      time.Time
	IntervalDays int
	DayOfMonth   int
	Dates        []time.Time
}

type GenerateInput struct {
	FromDate time.Time
	ToDate   time.Time
}
