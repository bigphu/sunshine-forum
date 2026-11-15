package identity

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Config của module identity, đọc bằng config.Load; binary nhúng nguyên khối, không
// thêm envPrefix. Field thời lượng bằng 0 (dựng tay trong test) thì lấy mặc định.
type Config struct {
	// Phiên trượt: không dùng tới trong khoảng này thì phải đăng nhập lại
	SessionTTL time.Duration `env:"IDENTITY_SESSION_TTL" envDefault:"336h"`
	// Cookie Secure phải tắt được khi dev chạy http
	CookieSecure bool   `env:"IDENTITY_COOKIE_SECURE" envDefault:"true"`
	CookieDomain string `env:"IDENTITY_COOKIE_DOMAIN"`

	// Quản trị viên đầu tiên, tạo lúc khởi động nếu chưa có tài khoản nào
	AdminEmail    string `env:"ADMIN_EMAIL"`
	AdminPassword string `env:"ADMIN_PASSWORD"`

	// Gốc của frontend; link trong thư mời / đặt lại mật khẩu trỏ về đây
	AppURL string `env:"IDENTITY_APP_URL" envDefault:"http://localhost:5180"`
	// Link mời hết hạn sau 3 ngày, link đặt lại sau 1 ngày (PRODUCT.md)
	InviteTTL time.Duration `env:"IDENTITY_INVITE_TTL" envDefault:"72h"`
	ResetTTL  time.Duration `env:"IDENTITY_RESET_TTL" envDefault:"24h"`
	// Xoá dữ liệu cá nhân của tài khoản đã khoá sau ngần này ngày (NFR-3.3)
	PurgeAfterDays int `env:"IDENTITY_PURGE_AFTER_DAYS" envDefault:"90"`
}

func (c Config) Validate() error {
	if c.SessionTTL < 0 || c.InviteTTL < 0 || c.ResetTTL < 0 || c.PurgeAfterDays < 0 {
		return errors.New("identity: durations must not be negative")
	}
	if (c.AdminEmail == "") != (c.AdminPassword == "") {
		return errors.New("identity: set both ADMIN_EMAIL and ADMIN_PASSWORD, or neither")
	}
	d := c.withDefaults()
	u, err := url.Parse(d.AppURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("identity: IDENTITY_APP_URL %q must be an http(s) URL without query or fragment", d.AppURL)
	}
	return nil
}

func (c Config) withDefaults() Config {
	if c.SessionTTL <= 0 {
		c.SessionTTL = 336 * time.Hour
	}
	if c.AppURL == "" {
		c.AppURL = "http://localhost:5180"
	}
	if c.InviteTTL <= 0 {
		c.InviteTTL = 72 * time.Hour
	}
	if c.ResetTTL <= 0 {
		c.ResetTTL = 24 * time.Hour
	}
	if c.PurgeAfterDays <= 0 {
		c.PurgeAfterDays = 90
	}
	return c
}
