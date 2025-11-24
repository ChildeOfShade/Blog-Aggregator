-- name: ListFeeds :many
SELECT
    f.id,
    f.name,
    f.url,
    u.name AS username
FROM feeds f
JOIN users u ON f.user_id = u.id;
