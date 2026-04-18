package recurringtask

import "time"

type Type string

const (
	TypeEveryNDays    Type = "every_n_days"
	TypeMonthlyDay    Type = "monthly_day"
	TypeSpecificDates Type = "specific_dates"
	TypeEvenDays      Type = "even_days"
	TypeOddDays       Type = "odd_days"
)

type Template struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Active      bool       `json:"active"`
	Recurrence  Recurrence `json:"recurrence"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Recurrence struct {
	Type         Type        `json:"type"`
	StartDate    time.Time   `json:"start_date,omitempty"`
	EndDate      time.Time   `json:"end_date,omitempty"`
	IntervalDays int         `json:"interval_days,omitempty"`
	DayOfMonth   int         `json:"day_of_month,omitempty"`
	Dates        []time.Time `json:"dates,omitempty"`
}

func (t Type) Valid() bool {
	switch t {
	case TypeEveryNDays, TypeMonthlyDay, TypeSpecificDates, TypeEvenDays, TypeOddDays:
		return true
	default:
		return false
	}
}
