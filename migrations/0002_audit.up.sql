CREATE TABLE audit_logs (
  id SERIAL PRIMARY KEY,
  task_id INT NOT NULL,
  operation TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);