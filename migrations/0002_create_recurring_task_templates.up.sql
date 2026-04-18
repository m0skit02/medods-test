CREATE TABLE IF NOT EXISTS recurring_task_templates (
	id BIGSERIAL PRIMARY KEY,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	active BOOLEAN NOT NULL DEFAULT TRUE,
	recurrence_type TEXT NOT NULL,
	start_date DATE,
	end_date DATE,
	interval_days INTEGER,
	day_of_month INTEGER,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT chk_recurring_task_templates_day_of_month
		CHECK (day_of_month IS NULL OR (day_of_month BETWEEN 1 AND 30)),
	CONSTRAINT chk_recurring_task_templates_interval_days
		CHECK (interval_days IS NULL OR interval_days > 0)
);

CREATE TABLE IF NOT EXISTS recurring_task_template_dates (
	template_id BIGINT NOT NULL REFERENCES recurring_task_templates (id) ON DELETE CASCADE,
	scheduled_date DATE NOT NULL,
	PRIMARY KEY (template_id, scheduled_date)
);

CREATE INDEX IF NOT EXISTS idx_recurring_task_templates_recurrence_type
	ON recurring_task_templates (recurrence_type);

CREATE INDEX IF NOT EXISTS idx_recurring_task_template_dates_scheduled_date
	ON recurring_task_template_dates (scheduled_date);
