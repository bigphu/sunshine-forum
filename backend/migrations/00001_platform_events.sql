-- Nhật ký event của outbox (goat/events). Event chỉ chứa ID, không có nội dung bài
-- hay dữ liệu cá nhân; job dọn dữ liệu xoá event cũ hơn 90 ngày (NFR-3.3).

-- +goose Up
CREATE SCHEMA IF NOT EXISTS platform;

CREATE TABLE platform.events (
    id             uuid        PRIMARY KEY,
    type           text        NOT NULL,
    aggregate_type text        NOT NULL,
    aggregate_id   uuid        NOT NULL,
    -- NULL: SystemActor (job định kỳ...)
    actor_id       uuid,
    occurred_at    timestamptz NOT NULL,
    payload        jsonb       NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX events_aggregate_idx ON platform.events (aggregate_type, aggregate_id, occurred_at);
CREATE INDEX events_type_idx ON platform.events (type, occurred_at);
CREATE INDEX events_aggregate_timeline_idx
    ON platform.events (aggregate_type, aggregate_id, occurred_at DESC, id DESC);

-- +goose Down
DROP TABLE platform.events;
DROP SCHEMA platform;
