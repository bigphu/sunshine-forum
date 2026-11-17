package main

import (
	"github.com/bigphu/goat/database"
	"github.com/bigphu/goat/logger"
	"github.com/bigphu/goat/server"

	"sunshine/internal/content"
	"sunshine/internal/identity"
	"sunshine/internal/notifications"
	"sunshine/internal/privacy"
	"sunshine/internal/screening"
)

// serverConfig chỉ liệt kê khối cấu hình binary này dùng. Tên biến env, mặc định và
// Validate nằm ở package sở hữu khối đó; config.Load tự gọi Validate.
type serverConfig struct {
	Log           logger.Config        // LOG_*
	HTTP          server.Config        // HTTP_*
	DB            database.Config      // DB_URL, PG*
	Privacy       privacy.Config       // PRIVACY_*
	Identity      identity.Config      // IDENTITY_*, ADMIN_*
	Screening     screening.Config     // SCREENING_*
	Content       content.Config       // CONTENT_*
	Notifications notifications.Config // NOTIFICATIONS_*
}
