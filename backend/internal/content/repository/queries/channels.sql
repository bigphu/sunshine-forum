-- name: ListChannels :many
SELECT * FROM content.channels ORDER BY position;

-- name: GetChannel :one
SELECT * FROM content.channels WHERE id = @id;

-- name: GetChannelBySlug :one
SELECT * FROM content.channels WHERE slug = @slug;
