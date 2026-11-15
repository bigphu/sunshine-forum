-- Module content: kênh, bài viết, bình luận hai cấp. Cột screening/ai_* lưu kết quả
-- sàng lọc gần nhất; search_text là tiêu đề + nội dung đã bỏ dấu, viết thường.

-- +goose Up
CREATE SCHEMA content;

CREATE TABLE content.channels (
    id            uuid PRIMARY KEY,
    slug          text NOT NULL UNIQUE,
    name          text NOT NULL,
    description   text NOT NULL DEFAULT '',
    official_only boolean NOT NULL DEFAULT false,
    position      int NOT NULL DEFAULT 0
);

CREATE TABLE content.posts (
    id                uuid PRIMARY KEY,
    channel_id        uuid NOT NULL REFERENCES content.channels (id),
    author_id         uuid NOT NULL,
    official          boolean NOT NULL DEFAULT false,
    title             text NOT NULL,
    body              text NOT NULL,
    search_text       text NOT NULL DEFAULT '',
    status            text NOT NULL CHECK (status IN ('published', 'pending', 'rejected', 'removed', 'auto_deleted', 'deleted')),
    pinned            boolean NOT NULL DEFAULT false,
    edited            boolean NOT NULL DEFAULT false,
    comment_count     int NOT NULL DEFAULT 0,
    -- Ban quản lý đã trả lời trong bài (dấu "Ban quản lý đã trả lời" ở bảng tin)
    official_replied  boolean NOT NULL DEFAULT false,
    screen_reason     text,
    ai_label          text,
    ai_score          real,
    model_version     text,
    flag_rule         int,
    ai_sentences      jsonb,
    needs_rescan      boolean NOT NULL DEFAULT false,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    status_changed_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX posts_feed_idx ON content.posts (channel_id, pinned DESC, created_at DESC, id DESC) WHERE status = 'published';
CREATE INDEX posts_all_feed_idx ON content.posts (pinned DESC, created_at DESC, id DESC) WHERE status = 'published';
CREATE INDEX posts_author_idx ON content.posts (author_id, created_at DESC);
CREATE INDEX posts_rescan_idx ON content.posts (created_at) WHERE needs_rescan;

CREATE TABLE content.comments (
    id                uuid PRIMARY KEY,
    post_id           uuid NOT NULL REFERENCES content.posts (id) ON DELETE CASCADE,
    parent_id         uuid REFERENCES content.comments (id) ON DELETE CASCADE,
    author_id         uuid NOT NULL,
    official          boolean NOT NULL DEFAULT false,
    body              text NOT NULL,
    status            text NOT NULL CHECK (status IN ('published', 'pending', 'rejected', 'removed', 'auto_deleted', 'deleted')),
    edited            boolean NOT NULL DEFAULT false,
    screen_reason     text,
    ai_label          text,
    ai_score          real,
    model_version     text,
    flag_rule         int,
    ai_sentences      jsonb,
    needs_rescan      boolean NOT NULL DEFAULT false,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    status_changed_at timestamptz NOT NULL DEFAULT now()
);
-- Cả chủ đề nạp bằng một truy vấn (bình luận chỉ hai cấp)
CREATE INDEX comments_thread_idx ON content.comments (post_id, created_at);
CREATE INDEX comments_author_idx ON content.comments (author_id, created_at DESC);
CREATE INDEX comments_rescan_idx ON content.comments (created_at) WHERE needs_rescan;

INSERT INTO content.channels (id, slug, name, description, official_only, position) VALUES
    ('00000000-0000-7000-8000-0000000000c1', 'thong-bao', 'Thông báo chung', 'Tin chính thức từ ban quản lý. Cư dân có thể bình luận.', true, 1),
    ('00000000-0000-7000-8000-0000000000c2', 'thao-luan', 'Thảo luận chung', 'Trao đổi giữa hàng xóm về mọi chuyện trong tòa nhà.', false, 2),
    ('00000000-0000-7000-8000-0000000000c3', 'bao-tri', 'Bảo trì', 'Báo hỏng hóc và yêu cầu sửa chữa. Ban quản lý trả lời trong bình luận.', false, 3),
    ('00000000-0000-7000-8000-0000000000c4', 'marketplace', 'Marketplace', 'Mua bán, trao đổi đồ dùng giữa cư dân.', false, 4);

INSERT INTO identity.permissions (code, description) VALUES
    ('content.read', 'Đọc kênh, bài viết và bình luận'),
    ('content.write', 'Đăng bài, bình luận, sửa và xoá nội dung của mình'),
    ('content.announce', 'Đăng bài trong Thông báo chung, trả lời chính thức'),
    ('content.pin', 'Ghim bài');

INSERT INTO identity.role_permissions (role_id, permission)
SELECT r.id, p.code FROM identity.roles r, identity.permissions p
WHERE p.code IN ('content.read', 'content.write')
   OR (p.code IN ('content.announce', 'content.pin') AND r.code IN ('manager', 'admin'));

-- +goose Down
DELETE FROM identity.permissions WHERE code LIKE 'content.%';
DROP SCHEMA content CASCADE;
