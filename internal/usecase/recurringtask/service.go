package recurringtask

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	recurringtaskdomain "example.com/taskservice/internal/domain/recurringtask"
	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo     Repository
	taskRepo TaskRepository
	now      func() time.Time
}

func NewService(repo Repository, taskRepo TaskRepository) *Service {
	return &Service{
		repo:     repo,
		taskRepo: taskRepo,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*recurringtaskdomain.Template, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	now := s.now()
	template := &recurringtaskdomain.Template{
		Title:       normalized.Title,
		Description: normalized.Description,
		Active:      normalized.Active,
		Recurrence:  toDomainRecurrence(normalized.Recurrence),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	created, err := s.repo.Create(ctx, template)
	if err != nil {
		return nil, err
	}

	if !created.Active {
		return created, nil
	}

	nowDate := normalizeDate(now)
	fromDate, toDate, ok := autoGenerationRange(created, nowDate)
	if !ok {
		return created, nil
	}

	if _, err := s.generateForRange(ctx, created, fromDate, toDate, now); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*recurringtaskdomain.Template, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*recurringtaskdomain.Template, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	template := &recurringtaskdomain.Template{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Active:      normalized.Active,
		Recurrence:  toDomainRecurrence(normalized.Recurrence),
		UpdatedAt:   s.now(),
	}

	return s.repo.Update(ctx, template)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]recurringtaskdomain.Template, error) {
	return s.repo.List(ctx)
}

func (s *Service) Generate(ctx context.Context, input GenerateInput) ([]taskdomain.Task, error) {
	fromDate, toDate, err := validateGenerateInput(input)
	if err != nil {
		return nil, err
	}

	templates, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	generated := make([]taskdomain.Task, 0)
	now := s.now()
	for i := range templates {
		template := &templates[i]
		if !template.Active {
			continue
		}

		tasks, err := s.generateForRange(ctx, template, fromDate, toDate, now)
		if err != nil {
			return nil, err
		}

		generated = append(generated, tasks...)
	}

	return generated, nil
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	recurrence, err := validateRecurrenceInput(input.Recurrence)
	if err != nil {
		return CreateInput{}, err
	}

	input.Recurrence = recurrence

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	recurrence, err := validateRecurrenceInput(input.Recurrence)
	if err != nil {
		return UpdateInput{}, err
	}

	input.Recurrence = recurrence

	return input, nil
}

func validateRecurrenceInput(input RecurrenceInput) (RecurrenceInput, error) {
	if !input.Type.Valid() {
		return RecurrenceInput{}, fmt.Errorf("%w: unsupported recurrence type", ErrInvalidInput)
	}

	switch input.Type {
	case recurringtaskdomain.TypeEveryNDays:
		return validateEveryNDaysInput(input)
	case recurringtaskdomain.TypeMonthlyDay:
		return validateMonthlyDayInput(input)
	case recurringtaskdomain.TypeSpecificDates:
		return validateSpecificDatesInput(input)
	case recurringtaskdomain.TypeEvenDays, recurringtaskdomain.TypeOddDays:
		return validateParityInput(input)
	default:
		return RecurrenceInput{}, fmt.Errorf("%w: unsupported recurrence type", ErrInvalidInput)
	}
}

func validateEveryNDaysInput(input RecurrenceInput) (RecurrenceInput, error) {
	if input.IntervalDays <= 0 {
		return RecurrenceInput{}, fmt.Errorf("%w: interval_days must be positive", ErrInvalidInput)
	}

	if input.StartDate.IsZero() {
		return RecurrenceInput{}, fmt.Errorf("%w: start_date is required", ErrInvalidInput)
	}

	input.StartDate = normalizeDate(input.StartDate)
	if !input.EndDate.IsZero() {
		input.EndDate = normalizeDate(input.EndDate)
	}
	if !input.EndDate.IsZero() && input.EndDate.Before(input.StartDate) {
		return RecurrenceInput{}, fmt.Errorf("%w: end_date must be greater than or equal to start_date", ErrInvalidInput)
	}

	input.DayOfMonth = 0
	input.Dates = nil

	return input, nil
}

func validateMonthlyDayInput(input RecurrenceInput) (RecurrenceInput, error) {
	if input.DayOfMonth < 1 || input.DayOfMonth > 30 {
		return RecurrenceInput{}, fmt.Errorf("%w: day_of_month must be between 1 and 30", ErrInvalidInput)
	}

	if !input.StartDate.IsZero() {
		input.StartDate = normalizeDate(input.StartDate)
	}
	if !input.EndDate.IsZero() {
		input.EndDate = normalizeDate(input.EndDate)
	}
	if !input.StartDate.IsZero() && !input.EndDate.IsZero() && input.EndDate.Before(input.StartDate) {
		return RecurrenceInput{}, fmt.Errorf("%w: end_date must be greater than or equal to start_date", ErrInvalidInput)
	}

	input.IntervalDays = 0
	input.Dates = nil

	return input, nil
}

func validateSpecificDatesInput(input RecurrenceInput) (RecurrenceInput, error) {
	if len(input.Dates) == 0 {
		return RecurrenceInput{}, fmt.Errorf("%w: dates must not be empty", ErrInvalidInput)
	}

	normalizedDates := make([]time.Time, 0, len(input.Dates))
	seen := make(map[string]struct{}, len(input.Dates))

	for _, date := range input.Dates {
		if date.IsZero() {
			return RecurrenceInput{}, fmt.Errorf("%w: date must not be empty", ErrInvalidInput)
		}

		normalized := normalizeDate(date)
		key := normalized.Format(time.DateOnly)
		if _, exists := seen[key]; exists {
			return RecurrenceInput{}, fmt.Errorf("%w: duplicate specific date", ErrInvalidInput)
		}

		seen[key] = struct{}{}
		normalizedDates = append(normalizedDates, normalized)
	}

	slices.SortFunc(normalizedDates, func(left, right time.Time) int {
		switch {
		case left.Before(right):
			return -1
		case left.After(right):
			return 1
		default:
			return 0
		}
	})

	input.StartDate = time.Time{}
	input.EndDate = time.Time{}
	input.IntervalDays = 0
	input.DayOfMonth = 0
	input.Dates = normalizedDates

	return input, nil
}

func validateParityInput(input RecurrenceInput) (RecurrenceInput, error) {
	if !input.StartDate.IsZero() {
		input.StartDate = normalizeDate(input.StartDate)
	}
	if !input.EndDate.IsZero() {
		input.EndDate = normalizeDate(input.EndDate)
	}
	if !input.StartDate.IsZero() && !input.EndDate.IsZero() && input.EndDate.Before(input.StartDate) {
		return RecurrenceInput{}, fmt.Errorf("%w: end_date must be greater than or equal to start_date", ErrInvalidInput)
	}

	input.IntervalDays = 0
	input.DayOfMonth = 0
	input.Dates = nil

	return input, nil
}

func toDomainRecurrence(input RecurrenceInput) recurringtaskdomain.Recurrence {
	return recurringtaskdomain.Recurrence{
		Type:         input.Type,
		StartDate:    input.StartDate,
		EndDate:      input.EndDate,
		IntervalDays: input.IntervalDays,
		DayOfMonth:   input.DayOfMonth,
		Dates:        append([]time.Time(nil), input.Dates...),
	}
}

func normalizeDate(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func validateGenerateInput(input GenerateInput) (time.Time, time.Time, error) {
	if input.FromDate.IsZero() {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: from_date is required", ErrInvalidInput)
	}

	if input.ToDate.IsZero() {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: to_date is required", ErrInvalidInput)
	}

	fromDate := normalizeDate(input.FromDate)
	toDate := normalizeDate(input.ToDate)
	if toDate.Before(fromDate) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: to_date must be greater than or equal to from_date", ErrInvalidInput)
	}

	return fromDate, toDate, nil
}

func matchesDate(template *recurringtaskdomain.Template, targetDate time.Time) bool {
	recurrence := template.Recurrence
	if !recurrence.StartDate.IsZero() && targetDate.Before(normalizeDate(recurrence.StartDate)) {
		return false
	}
	if !recurrence.EndDate.IsZero() && targetDate.After(normalizeDate(recurrence.EndDate)) {
		return false
	}

	switch template.Recurrence.Type {
	case recurringtaskdomain.TypeEveryNDays:
		return matchesEveryNDays(template.Recurrence, targetDate)
	case recurringtaskdomain.TypeMonthlyDay:
		return targetDate.Day() == template.Recurrence.DayOfMonth
	case recurringtaskdomain.TypeSpecificDates:
		for _, date := range template.Recurrence.Dates {
			if normalizeDate(date).Equal(targetDate) {
				return true
			}
		}
		return false
	case recurringtaskdomain.TypeEvenDays:
		return targetDate.Day()%2 == 0
	case recurringtaskdomain.TypeOddDays:
		return targetDate.Day()%2 != 0
	default:
		return false
	}
}

func matchesEveryNDays(recurrence recurringtaskdomain.Recurrence, targetDate time.Time) bool {
	startDate := normalizeDate(recurrence.StartDate)
	if startDate.IsZero() || targetDate.Before(startDate) {
		return false
	}

	daysSinceStart := int(targetDate.Sub(startDate).Hours() / 24)
	return daysSinceStart%recurrence.IntervalDays == 0
}

func autoGenerationRange(template *recurringtaskdomain.Template, nowDate time.Time) (time.Time, time.Time, bool) {
	if template.Recurrence.Type == recurringtaskdomain.TypeSpecificDates {
		var (
			fromDate time.Time
			toDate   time.Time
			found    bool
		)

		for _, date := range template.Recurrence.Dates {
			normalized := normalizeDate(date)
			if normalized.Before(nowDate) {
				continue
			}

			if !found || normalized.Before(fromDate) {
				fromDate = normalized
			}
			if !found || normalized.After(toDate) {
				toDate = normalized
			}
			found = true
		}

		if !found {
			return time.Time{}, time.Time{}, false
		}

		return fromDate, toDate, true
	}

	if template.Recurrence.EndDate.IsZero() {
		return time.Time{}, time.Time{}, false
	}

	fromDate := nowDate
	if !template.Recurrence.StartDate.IsZero() && template.Recurrence.StartDate.After(fromDate) {
		fromDate = normalizeDate(template.Recurrence.StartDate)
	}

	toDate := normalizeDate(template.Recurrence.EndDate)
	if toDate.Before(fromDate) {
		return time.Time{}, time.Time{}, false
	}

	return fromDate, toDate, true
}

func (s *Service) generateForRange(ctx context.Context, template *recurringtaskdomain.Template, fromDate time.Time, toDate time.Time, createdAt time.Time) ([]taskdomain.Task, error) {
	matchingDates := collectMatchingDates(template, fromDate, toDate)
	generated := make([]taskdomain.Task, 0, len(matchingDates))
	for _, scheduledFor := range matchingDates {
		task, err := s.taskRepo.CreateGeneratedFromTemplate(ctx, template, scheduledFor, createdAt)
		if err != nil {
			return nil, err
		}
		if task == nil {
			continue
		}

		generated = append(generated, *task)
	}

	return generated, nil
}

func collectMatchingDates(template *recurringtaskdomain.Template, fromDate time.Time, toDate time.Time) []time.Time {
	switch template.Recurrence.Type {
	case recurringtaskdomain.TypeSpecificDates:
		dates := make([]time.Time, 0, len(template.Recurrence.Dates))
		for _, date := range template.Recurrence.Dates {
			normalized := normalizeDate(date)
			if normalized.Before(fromDate) || normalized.After(toDate) {
				continue
			}
			if !matchesDate(template, normalized) {
				continue
			}
			dates = append(dates, normalized)
		}
		return dates
	default:
		dates := make([]time.Time, 0)
		for current := fromDate; !current.After(toDate); current = current.AddDate(0, 0, 1) {
			if matchesDate(template, current) {
				dates = append(dates, current)
			}
		}
		return dates
	}
}
