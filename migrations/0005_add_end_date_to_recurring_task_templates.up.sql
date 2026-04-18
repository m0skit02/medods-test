ALTER TABLE recurring_task_templates
	ADD COLUMN IF NOT EXISTS end_date DATE;
