package main

import (
	"github.com/bigphu/goat/database"
	"github.com/bigphu/goat/jobs"
	"github.com/bigphu/goat/logger"
	"github.com/bigphu/goat/mail"

	"sunshine/internal/content"
	"sunshine/internal/identity"
	"sunshine/internal/notifications"
	"sunshine/internal/privacy"
)

type workerConfig struct {
	Log           logger.Config        // LOG_*
	DB            database.Config      // DB_URL, PG*
	Jobs          jobs.Config          // JOBS_*
	Mail          mail.Config          // MAIL_*
	Privacy       privacy.Config       // PRIVACY_*
	Identity      identity.Config      // IDENTITY_*
	Content       content.Config       // CONTENT_*
	Notifications notifications.Config // NOTIFICATIONS_*
	// Event của outbox được xoá sau ngần này ngày (NFR-3.3)
	EventsPurgeAfterDays int `env:"EVENTS_PURGE_AFTER_DAYS" envDefault:"90"`
}
