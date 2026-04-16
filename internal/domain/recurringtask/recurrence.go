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

type Recurrence struct {
	Type         Type
	StartDate    time.Time
	IntervalDays int
	DayOfMonth   int
	Dates        []time.Time
}

func (t Type) Valid() bool {
	switch t {
	case TypeEveryNDays, TypeMonthlyDay, TypeSpecificDates, TypeEvenDays, TypeOddDays:
		return true
	default:
		return false
	}
}
