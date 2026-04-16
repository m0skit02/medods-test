package recurringtask

import "time"

type Template struct {
	ID          int64
	Title       string
	Description string
	Active      bool
	Recurrence  Recurrence
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
