package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	recurringtaskdomain "example.com/taskservice/internal/domain/recurringtask"
	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (title, description, status, created_at, updated_at, scheduled_for)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, title, description, status, scheduled_for, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query, task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt, nullableTaskDate(task.Date))
	created, err := scanTask(row)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, scheduled_for, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET title = $1,
			description = $2,
			status = $3,
			scheduled_for = $4,
			updated_at = $5
		WHERE id = $6
		RETURNING id, title, description, status, scheduled_for, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query, task.Title, task.Description, task.Status, nullableTaskDate(task.Date), task.UpdatedAt, task.ID)
	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, scheduled_for, created_at, updated_at
		FROM tasks
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *Repository) CreateGeneratedFromTemplate(ctx context.Context, template *recurringtaskdomain.Template, scheduledFor time.Time, createdAt time.Time) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (title, description, status, created_at, updated_at, recurring_template_id, scheduled_for)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (recurring_template_id, scheduled_for) DO NOTHING
		RETURNING id, title, description, status, scheduled_for, created_at, updated_at
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		template.Title,
		template.Description,
		taskdomain.StatusNew,
		createdAt,
		createdAt,
		template.ID,
		scheduledFor.Format(time.DateOnly),
	)

	created, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return created, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task         taskdomain.Task
		status       string
		scheduledFor *time.Time
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&scheduledFor,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)
	if scheduledFor != nil {
		task.Date = normalizeTaskDate(*scheduledFor)
	}

	return &task, nil
}

func nullableTaskDate(value time.Time) any {
	if value.IsZero() {
		return nil
	}

	return value.Format(time.DateOnly)
}

func normalizeTaskDate(value time.Time) time.Time {
	year, month, day := value.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
