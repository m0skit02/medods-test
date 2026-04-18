ALTER TABLE tasks
	ADD COLUMN IF NOT EXISTS recurring_template_id BIGINT REFERENCES recurring_task_templates (id) ON DELETE SET NULL,
	ADD COLUMN IF NOT EXISTS scheduled_for DATE;

CREATE UNIQUE INDEX IF NOT EXISTS uq_tasks_recurring_template_id_scheduled_for
	ON tasks (recurring_template_id, scheduled_for);
