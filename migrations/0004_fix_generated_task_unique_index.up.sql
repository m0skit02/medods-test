DROP INDEX IF EXISTS uq_tasks_recurring_template_id_scheduled_for;

CREATE UNIQUE INDEX IF NOT EXISTS uq_tasks_recurring_template_id_scheduled_for
	ON tasks (recurring_template_id, scheduled_for);
