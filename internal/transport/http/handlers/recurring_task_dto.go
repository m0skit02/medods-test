package handlers

import (
	"time"

	recurringtaskdomain "example.com/taskservice/internal/domain/recurringtask"
)

type recurringTaskMutationDTO struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Active      bool                   `json:"active"`
	Recurrence  recurringRecurrenceDTO `json:"recurrence"`
}

type recurringTaskDTO struct {
	ID          int64                  `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Active      bool                   `json:"active"`
	Recurrence  recurringRecurrenceDTO `json:"recurrence"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type recurringRecurrenceDTO struct {
	Type         recurringtaskdomain.Type `json:"type"`
	StartDate    time.Time                `json:"start_date,omitempty"`
	EndDate      time.Time                `json:"end_date,omitempty"`
	IntervalDays int                      `json:"interval_days,omitempty"`
	DayOfMonth   int                      `json:"day_of_month,omitempty"`
	Dates        []time.Time              `json:"dates,omitempty"`
}

type recurringTaskGenerateDTO struct {
	FromDate time.Time `json:"from_date"`
	ToDate   time.Time `json:"to_date"`
}

type recurringTaskGenerateResponseDTO struct {
	FromDate       time.Time `json:"from_date"`
	ToDate         time.Time `json:"to_date"`
	GeneratedCount int       `json:"generated_count"`
	Tasks          []taskDTO `json:"tasks"`
}

func newRecurringTaskDTO(template *recurringtaskdomain.Template) recurringTaskDTO {
	return recurringTaskDTO{
		ID:          template.ID,
		Title:       template.Title,
		Description: template.Description,
		Active:      template.Active,
		Recurrence: recurringRecurrenceDTO{
			Type:         template.Recurrence.Type,
			StartDate:    template.Recurrence.StartDate,
			EndDate:      template.Recurrence.EndDate,
			IntervalDays: template.Recurrence.IntervalDays,
			DayOfMonth:   template.Recurrence.DayOfMonth,
			Dates:        append([]time.Time(nil), template.Recurrence.Dates...),
		},
		CreatedAt: template.CreatedAt,
		UpdatedAt: template.UpdatedAt,
	}
}
