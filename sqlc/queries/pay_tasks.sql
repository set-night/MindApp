-- name: GetAvailablePayTasks :many
SELECT pt.*, COUNT(ptc.id)::int AS completed_count
FROM pay_tasks pt
LEFT JOIN pay_task_completions ptc ON ptc.task_id = pt.id
WHERE (pt.time_limit IS NULL OR pt.time_limit > NOW())
GROUP BY pt.id
HAVING (pt.max_people IS NULL OR COUNT(ptc.id) < pt.max_people);

-- name: GetPayTaskByID :one
SELECT pt.*, COUNT(ptc.id)::int AS completed_count
FROM pay_tasks pt
LEFT JOIN pay_task_completions ptc ON ptc.task_id = pt.id
WHERE pt.id = $1
GROUP BY pt.id;

-- name: CheckPayTaskCompletion :one
SELECT EXISTS(
    SELECT 1 FROM pay_task_completions WHERE task_id = $1 AND user_id = $2
) AS completed;

-- name: CreatePayTaskCompletion :exec
INSERT INTO pay_task_completions (task_id, user_id) VALUES ($1, $2);

-- name: GetAvailablePayTasksForUser :many
SELECT pt.*, COUNT(ptc.id)::int AS completed_count
FROM pay_tasks pt
LEFT JOIN pay_task_completions ptc ON ptc.task_id = pt.id
WHERE (pt.time_limit IS NULL OR pt.time_limit > NOW())
  AND pt.id NOT IN (SELECT ptc2.task_id FROM pay_task_completions ptc2 WHERE ptc2.user_id = $1)
GROUP BY pt.id
HAVING (pt.max_people IS NULL OR COUNT(ptc.id) < pt.max_people);
