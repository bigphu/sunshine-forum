// cmd/seed tạo dữ liệu demo: năm tài khoản (mật khẩu SEED_PASSWORD) và vài bài, bình luận
// qua đúng use case của app, nên nội dung đi qua sàng lọc và phát event như thật. Mọi
// người và bài ở đây là dữ liệu mẫu, tên kết thúc bằng "(mẫu)". Chạy lại thì bỏ qua phần
// đã có.
//
//	make seed   // hoặc: SEED_PASSWORD=... go run ./cmd/seed
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bigphu/goat/auth"
	"github.com/bigphu/goat/config"
	"github.com/bigphu/goat/database"
	"github.com/bigphu/goat/events"
	"github.com/bigphu/goat/jobs"
	"github.com/bigphu/goat/logger"
	"github.com/google/uuid"

	"sunshine/internal/content"
	"sunshine/internal/identity"
	"sunshine/internal/moderation"
	"sunshine/internal/notifications"
	"sunshine/internal/penalties"
	"sunshine/internal/privacy"
	"sunshine/internal/screening"
)

type seedConfig struct {
	Log       logger.Config
	DB        database.Config
	Privacy   privacy.Config
	Identity  identity.Config
	Screening screening.Config
	Password  string `env:"SEED_PASSWORD"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var people = []struct{ key, email, name, unit, role string }{
	{"admin", "admin@sunshine.test", "Chủ tòa nhà (mẫu)", "", "admin"},
	{"manager", "manager@sunshine.test", "Ban quản lý (mẫu)", "", "manager"},
	{"lan", "lan@sunshine.test", "Nguyễn Thị Lan (mẫu)", "A-1203", "resident"},
	{"minh", "minh@sunshine.test", "Trần Văn Minh (mẫu)", "B-0807", "resident"},
	{"hoa", "hoa@sunshine.test", "Lê Thu Hoa (mẫu)", "A-0405", "resident"},
}

func run() error {
	var cfg seedConfig
	if err := config.Load(&cfg); err != nil {
		return err
	}
	if len(cfg.Password) < 8 {
		return errors.New("seed: set SEED_PASSWORD (at least 8 characters)")
	}
	log := logger.New(os.Stdout, cfg.Log)
	slog.SetDefault(log)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer database.Close(pool)
	cipher, err := privacy.New(cfg.Privacy)
	if err != nil {
		return err
	}
	insertClient, err := jobs.NewInsertClient(pool, log)
	if err != nil {
		return err
	}
	// Cùng registry như API: event được xếp cho đủ subscriber, worker giao sau
	registry := events.NewRegistry()
	outbox := events.NewOutbox(registry, insertClient)
	identityMod, err := identity.New(ctx, identity.Deps{Pool: pool, Outbox: outbox, Jobs: jobs.NewRiver(insertClient), Cipher: cipher, Config: cfg.Identity})
	if err != nil {
		return err
	}
	screeningMod, err := screening.New(screening.Deps{Pool: pool, Outbox: outbox, Config: cfg.Screening})
	if err != nil {
		return err
	}
	penaltiesMod, err := penalties.New(penalties.Deps{Pool: pool, Outbox: outbox})
	if err != nil {
		return err
	}
	contentMod, err := content.New(content.Deps{Pool: pool, Outbox: outbox, Screener: screeningMod.Screener(),
		Accounts: identityMod.AccountReader(), Standing: penaltiesMod.Standing()})
	if err != nil {
		return err
	}
	moderationMod, err := moderation.New(moderation.Deps{Pool: pool, Outbox: outbox, Content: contentMod.Reader(),
		Accounts: identityMod.AccountReader(), Standing: penaltiesMod.Standing(), Cipher: cipher})
	if err != nil {
		return err
	}
	notificationsMod, err := notifications.New(notifications.Deps{Pool: pool, Content: contentMod.Reader(), Accounts: identityMod.AccountReader()})
	if err != nil {
		return err
	}
	contentMod.Subscribe(registry)
	moderationMod.Subscribe(registry)
	penaltiesMod.Subscribe(registry)
	notificationsMod.Subscribe(registry)

	ids := map[string]uuid.UUID{}
	for _, p := range people {
		id, err := identityMod.SeedAccount(ctx, p.email, p.name, p.unit, p.role, cfg.Password)
		if err != nil {
			return fmt.Errorf("seed %s: %w", p.email, err)
		}
		ids[p.key] = id
	}
	as := func(key string) (context.Context, error) {
		actor, err := identityMod.LoadActor(ctx, ids[key])
		if err != nil {
			return nil, err
		}
		return auth.WithActor(ctx, actor), nil
	}
	svc := contentMod.Service()

	adminCtx, err := as("admin")
	if err != nil {
		return err
	}
	feed, err := svc.Feed(adminCtx, "", nil)
	if err != nil {
		return err
	}
	if len(feed.Items)+len(feed.Pinned) > 0 {
		log.Info("seed: content already present, accounts checked")
		return nil
	}
	chans, err := svc.Channels(adminCtx)
	if err != nil {
		return err
	}
	ch := map[string]uuid.UUID{}
	for _, c := range chans {
		ch[c.Slug] = c.ID
	}

	type step func() error
	post := func(who, slug, title, body string, out *uuid.UUID) step {
		return func() error {
			c, err := as(who)
			if err != nil {
				return err
			}
			s, err := svc.CreatePost(c, ch[slug], title, body)
			if out != nil {
				*out = s.PostID
			}
			return err
		}
	}
	comment := func(who string, postID *uuid.UUID, parent *uuid.UUID, body string, out *uuid.UUID) step {
		return func() error {
			c, err := as(who)
			if err != nil {
				return err
			}
			s, err := svc.CreateComment(c, *postID, parent, body)
			if out != nil && s.CommentID != nil {
				*out = *s.CommentID
			}
			return err
		}
	}
	var water, light, parking, q1 uuid.UUID
	// Từ khoá chặn mẫu (Quy tắc 6), để thử nhóm "Từ khóa chặn" trong hàng chờ
	blocks := func() error {
		for _, t := range []string{"vay nhanh", "lãi cực thấp"} {
			if _, err := screeningMod.Service().AddBlock(adminCtx, t, 6); err != nil {
				return err
			}
		}
		return nil
	}
	steps := []step{
		blocks,
		post("minh", "marketplace", "Hỗ trợ tài chính cho cư dân",
			"Vay nhanh trong ngày, lãi cực thấp, không cần thế chấp. Nhắn tin riêng để được tư vấn.", nil),
		post("manager", "thong-bao", "Lịch cắt nước bảo trì bể chứa tháng 11",
			"Ban quản lý thông báo: tòa A và B sẽ tạm ngưng cấp nước từ 8:00 đến 12:00 sáng thứ Bảy để vệ sinh bể chứa. Cư dân vui lòng trữ nước trước. Xin cảm ơn sự hợp tác của cư dân.", &water),
		func() error {
			c, err := as("manager")
			if err != nil {
				return err
			}
			return svc.SetPinned(c, water, true)
		},
		comment("lan", &water, nil, "Cảm ơn ban quản lý đã báo sớm. Thứ Bảy tuần này hay tuần sau ạ?", &q1),
		comment("manager", &water, &q1, "Thứ Bảy tuần này chị nhé, ngày 21/11.", nil),
		post("minh", "bao-tri", "Đèn hành lang tầng 8 tòa B bị chập chờn",
			"Đèn ở đoạn gần thang máy tầng 8 tòa B nhấp nháy mấy hôm nay, buổi tối đi lại hơi khó nhìn. Nhờ ban quản lý kiểm tra giúp.", &light),
		comment("manager", &light, nil, "Đã ghi nhận, kỹ thuật sẽ thay bóng trong chiều nay.", nil),
		post("hoa", "thao-luan", "Gửi xe máy ở hầm B2 có ai thấy chật không?",
			"Dạo này buổi tối về muộn gần như không còn chỗ ở hầm B2. Mọi người có đề xuất gì để sắp xếp lại không?", &parking),
		comment("lan", &parking, nil, "Mình cũng thấy vậy, chắc nên kẻ lại vạch cho xe máy.", nil),
		// Bình luận này bị bộ lọc (điểm mẫu) giữ lại, để hàng chờ có sẵn một mục
		comment("minh", &parking, nil, "Ai để xe lấn chỗ thì đúng là đồ khốn, ngu không chịu được.", nil),
		post("lan", "marketplace", "Pass lại nồi chiên không dầu còn mới",
			"Nồi chiên 5 lít dùng được 3 tháng, còn bảo hành. Ai cần nhắn mình qua diễn đàn nhé.", nil),
	}
	for i, s := range steps {
		if err := s(); err != nil {
			return fmt.Errorf("seed: step %d: %w", i+1, err)
		}
	}
	log.Info("seed: demo data created; start the worker to deliver its events", slog.Int("accounts", len(people)))
	return nil
}
