-- name: ListRoles :many
SELECT * FROM identity.roles ORDER BY code DESC;

-- name: GetRole :one
SELECT * FROM identity.roles WHERE id = @id;

-- name: GetRoleByCode :one
SELECT * FROM identity.roles WHERE code = @code;

-- name: RolePermissions :many
SELECT permission FROM identity.role_permissions WHERE role_id = @role_id ORDER BY permission;
