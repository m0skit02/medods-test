package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	recurringtaskdomain "example.com/taskservice/internal/domain/recurringtask"
)

type RecurringTaskRepository struct {
	pool *pgxpool.Pool
}

func NewRecurringTaskRepository(pool *pgxpool.Pool) *RecurringTaskRepository {
	return &RecurringTaskRepository{pool: pool}
}

func (r *RecurringTaskRepository) Create(ctx context.Context, template *recurringtaskdomain.Template) (*recurringtaskdomain.Template, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	const query = `
		INSERT INTO recurring_task_templates (
			title, description, active, recurrence_type, start_date, end_date, interval_days, day_of_month, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, title, description, active, recurrence_type, start_date, end_date, interval_days, day_of_month, created_at, updated_at
	`

	row := tx.QueryRow(
		ctx,
		query,
		template.Title,
		template.Description,
		template.Active,
		template.Recurrence.Type,
		nullableDate(template.Recurrence.StartDate),
		nullableDate(template.Recurrence.EndDate),
		nullableInt(template.Recurrence.IntervalDays),
		nullableInt(template.Recurrence.DayOfMonth),
		template.CreatedAt,
		template.UpdatedAt,
	)

	created, err := scanRecurringTemplate(row)
	if err != nil {
		return nil, err
	}

	if err := r.replaceDates(ctx, tx, created.ID, template.Recurrence.Dates); err != nil {
		return nil, err
	}

	created.Recurrence.Dates = cloneDates(template.Recurrence.Dates)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return created, nil
}

func (r *RecurringTaskRepository) GetByID(ctx context.Context, id int64) (*recurringtaskdomain.Template, error) {
	const query = `
		SELECT id, title, description, active, recurrence_type, start_date, end_date, interval_days, day_of_month, created_at, updated_at
		FROM recurring_task_templates
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	template, err := scanRecurringTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, recurringtaskdomain.ErrNotFound
		}

		return nil, err
	}

	dates, err := r.listDatesByTemplateIDs(ctx, []int64{id})
	if err != nil {
		return nil, err
	}

	template.Recurrence.Dates = dates[id]

	return template, nil
}

func (r *RecurringTaskRepository) Update(ctx context.Context, template *recurringtaskdomain.Template) (*recurringtaskdomain.Template, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	const query = `
		UPDATE recurring_task_templates
		SET title = $1,
			description = $2,
			active = $3,
			recurrence_type = $4,
			start_date = $5,
			end_date = $6,
			interval_days = $7,
			day_of_month = $8,
			updated_at = $9
		WHERE id = $10
		RETURNING id, title, description, active, recurrence_type, start_date, end_date, interval_days, day_of_month, created_at, updated_at
	`

	row := tx.QueryRow(
		ctx,
		query,
		template.Title,
		template.Description,
		template.Active,
		template.Recurrence.Type,
		nullableDate(template.Recurrence.StartDate),
		nullableDate(template.Recurrence.EndDate),
		nullableInt(template.Recurrence.IntervalDays),
		nullableInt(template.Recurrence.DayOfMonth),
		template.UpdatedAt,
		template.ID,
	)

	updated, err := scanRecurringTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, recurringtaskdomain.ErrNotFound
		}

		return nil, err
	}

	if err := r.replaceDates(ctx, tx, updated.ID, template.Recurrence.Dates); err != nil {
		return nil, err
	}

	updated.Recurrence.Dates = cloneDates(template.Recurrence.Dates)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return updated, nil
}

func (r *RecurringTaskRepository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM recurring_task_templates WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return recurringtaskdomain.ErrNotFound
	}

	return nil
}

func (r *RecurringTaskRepository) List(ctx context.Context) ([]recurringtaskdomain.Template, error) {
	const query = `
		SELECT id, title, description, active, recurrence_type, start_date, end_date, interval_days, day_of_month, created_at, updated_at
		FROM recurring_task_templates
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := make([]recurringtaskdomain.Template, 0)
	ids := make([]int64, 0)
	for rows.Next() {
		template, err := scanRecurringTemplate(rows)
		if err != nil {
			return nil, err
		}

		templates = append(templates, *template)
		ids = append(ids, template.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return templates, nil
	}

	datesByTemplateID, err := r.listDatesByTemplateIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	for i := range templates {
		templates[i].Recurrence.Dates = datesByTemplateID[templates[i].ID]
	}

	return templates, nil
}

func (r *RecurringTaskRepository) replaceDates(ctx context.Context, tx pgx.Tx, templateID int64, dates []time.Time) error {
	if _, err := tx.Exec(ctx, `DELETE FROM recurring_task_template_dates WHERE template_id = $1`, templateID); err != nil {
		return err
	}

	if len(dates) == 0 {
		return nil
	}

	for _, date := range dates {
		if _, err := tx.Exec(
			ctx,
			`INSERT INTO recurring_task_template_dates (template_id, scheduled_date) VALUES ($1, $2)`,
			templateID,
			date.Format(time.DateOnly),
		); err != nil {
			return err
		}
	}

	return nil
}

func (r *RecurringTaskRepository) listDatesByTemplateIDs(ctx context.Context, ids []int64) (map[int64][]time.Time, error) {
	rows, err := r.pool.Query(
		ctx,
		`
			SELECT template_id, scheduled_date
			FROM recurring_task_template_dates
			WHERE template_id = ANY($1)
			ORDER BY template_id, scheduled_date
		`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	datesByTemplateID := make(map[int64][]time.Time, len(ids))
	for rows.Next() {
		var (
			templateID int64
			date       time.Time
		)

		if err := rows.Scan(&templateID, &date); err != nil {
			return nil, err
		}

		datesByTemplateID[templateID] = append(datesByTemplateID[templateID], normalizeUTCDate(date))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return datesByTemplateID, nil
}

type recurringTemplateScanner interface {
	Scan(dest ...any) error
}

func scanRecurringTemplate(scanner recurringTemplateScanner) (*recurringtaskdomain.Template, error) {
	var (
		template     recurringtaskdomain.Template
		recType      string
		startDate    *time.Time
		endDate      *time.Time
		intervalDays *int
		dayOfMonth   *int
	)

	if err := scanner.Scan(
		&template.ID,
		&template.Title,
		&template.Description,
		&template.Active,
		&recType,
		&startDate,
		&endDate,
		&intervalDays,
		&dayOfMonth,
		&template.CreatedAt,
		&template.UpdatedAt,
	); err != nil {
		return nil, err
	}

	template.Recurrence.Type = recurringtaskdomain.Type(recType)
	if startDate != nil {
		template.Recurrence.StartDate = normalizeUTCDate(*startDate)
	}
	if endDate != nil {
		template.Recurrence.EndDate = normalizeUTCDate(*endDate)
	}
	if intervalDays != nil {
		template.Recurrence.IntervalDays = *intervalDays
	}
	if dayOfMonth != nil {
		template.Recurrence.DayOfMonth = *dayOfMonth
	}

	return &template, nil
}

func nullableDate(value time.Time) any {
	if value.IsZero() {
		return nil
	}

	return value.Format(time.DateOnly)
}

func nullableInt(value int) any {
	if value == 0 {
		return nil
	}

	return value
}

func cloneDates(dates []time.Time) []time.Time {
	if len(dates) == 0 {
		return nil
	}

	cloned := make([]time.Time, len(dates))
	copy(cloned, dates)
	return cloned
}

func normalizeUTCDate(value time.Time) time.Time {
	year, month, day := value.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
