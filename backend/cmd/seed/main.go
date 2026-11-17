// cmd/seed tạo dữ liệu demo: mười tài khoản (mật khẩu SEED_PASSWORD), khoảng hai mươi bài
// và hai mươi lăm bình luận trong bốn kênh, đi qua đúng use case của app. Vì vậy nội dung
// được bộ lọc thật chấm điểm (SCREENING_MODEL_URL): mô hình quyết định cái nào được đăng,
// cái nào bị giữ lại, kể cả khi nó sai.
//
// Mỗi nội dung có "ý định" do người viết seed gán (sạch, thô tục, thù ghét, từ khóa chặn,
// tiếng nước ngoài). Sau đó một lượt quản lý xử lý như người thật: duyệt nội dung sạch bị
// giữ nhầm, từ chối nội dung xấu bị giữ, gỡ nội dung xấu lọt lưới, chừa vài mục trong hàng
// chờ, và một cư dân yêu cầu xem xét lại. Lượt này cần worker đang chạy (event giao nội
// dung bị giữ vào hàng chờ); không có worker thì seed chỉ tạo nội dung.
//
// Thời gian của bài và bình luận được lùi về trong tuần qua để bảng tin trông như thật.
// Mọi người và bài là dữ liệu mẫu, tên kết thúc bằng "(mẫu)". Chạy lại thì bỏ qua.
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
	"time"

	"github.com/bigphu/goat/auth"
	"github.com/bigphu/goat/config"
	"github.com/bigphu/goat/database"
	"github.com/bigphu/goat/events"
	"github.com/bigphu/goat/jobs"
	"github.com/bigphu/goat/logger"
	"github.com/google/uuid"

	"sunshine/internal/content"
	contentc "sunshine/internal/content/contract"
	"sunshine/internal/identity"
	"sunshine/internal/moderation"
	moderationc "sunshine/internal/moderation/contract"
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
	{"tuan", "tuan@sunshine.test", "Phạm Minh Tuấn (mẫu)", "C-1508", "resident"},
	{"thao", "thao@sunshine.test", "Võ Thanh Thảo (mẫu)", "B-1102", "resident"},
	{"khoa", "khoa@sunshine.test", "Đặng Khoa (mẫu)", "C-0310", "resident"},
	{"ngoc", "ngoc@sunshine.test", "Bùi Bảo Ngọc (mẫu)", "A-0907", "resident"},
	{"kevin", "kevin@sunshine.test", "Kevin Nguyễn (mẫu)", "B-0511", "resident"},
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

	// Từ khoá chặn mẫu (Quy tắc 6), để thử nhóm "Từ khóa chặn" trong hàng chờ
	for _, t := range []string{"vay nhanh", "lãi cực thấp"} {
		if _, err := screeningMod.Service().AddBlock(adminCtx, t, 6); err != nil {
			return err
		}
	}

	// 1. Đăng nội dung theo kế hoạch; mô hình quyết định đăng hay giữ lại
	items := make([]created, 0, len(posts)+len(comments))
	postIDs := map[string]uuid.UUID{}
	commentIDs := map[string]uuid.UUID{}
	for _, p := range posts {
		c, err := as(p.who)
		if err != nil {
			return err
		}
		s, err := svc.CreatePost(c, ch[p.channel], p.title, p.body)
		if err != nil {
			return fmt.Errorf("seed post %q: %w", p.key, err)
		}
		postIDs[p.key] = s.PostID
		if p.pin {
			if err := svc.SetPinned(c, s.PostID, true); err != nil {
				return err
			}
		}
		items = append(items, created{ref: contentc.Ref{Type: "post", ID: s.PostID}, status: s.Status, plan: p.plan})
	}
	for _, m := range comments {
		c, err := as(m.who)
		if err != nil {
			return err
		}
		var parent *uuid.UUID
		if m.parent != "" {
			id := commentIDs[m.parent]
			parent = &id
		}
		s, err := svc.CreateComment(c, postIDs[m.post], parent, m.body)
		if err != nil {
			return fmt.Errorf("seed comment on %q: %w", m.post, err)
		}
		if s.CommentID == nil {
			return fmt.Errorf("seed comment on %q: no id returned", m.post)
		}
		if m.key != "" {
			commentIDs[m.key] = *s.CommentID
		}
		items = append(items, created{ref: contentc.Ref{Type: "comment", ID: *s.CommentID}, status: s.Status, plan: m.plan})
	}
	held := map[uuid.UUID]bool{}
	for _, x := range items {
		if x.status == "pending" {
			held[x.ref.ID] = true
		}
	}
	log.Info("seed: content created", slog.Int("items", len(items)), slog.Int("held_by_filter", len(held)))

	// 2. Lùi thời gian cho giống một tuần sinh hoạt thật (chỉ dữ liệu demo)
	for _, x := range items {
		table := "content.posts"
		if x.ref.Type == "comment" {
			table = "content.comments"
		}
		if _, err := pool.Exec(ctx, "UPDATE "+table+" SET created_at = now() - make_interval(mins => $1) WHERE id = $2",
			int(x.ago.Minutes()), x.ref.ID); err != nil {
			return fmt.Errorf("seed backdate: %w", err)
		}
	}

	// 3. Lượt quản lý: chờ worker đưa nội dung bị giữ vào hàng chờ rồi xử lý như người thật
	managerCtx, err := as("manager")
	if err != nil {
		return err
	}
	modSvc := moderationMod.Service()
	inQueue := func() (int, error) {
		q, err := modSvc.Queue(managerCtx)
		if err != nil {
			return 0, err
		}
		n := 0
		for _, it := range q.Items {
			if held[it.Item.Ref.ID] {
				n++
			}
		}
		return n, nil
	}
	ready, err := waitFor(ctx, time.Minute, func() (bool, error) {
		n, err := inQueue()
		return n == len(held), err
	})
	if err != nil {
		return err
	}
	if !ready {
		log.Warn("seed: the worker did not deliver held content within a minute, so the manager pass was skipped. " +
			"Start the worker (make up) and handle the queue in the app")
		return nil
	}
	decided := map[string]int{}
	for _, x := range items {
		if x.leave {
			continue
		}
		toxic := x.intent != intentClean && x.intent != intentForeign
		switch {
		case x.status == "pending" && !toxic:
			if _, err := modSvc.Decide(managerCtx, []contentc.Ref{x.ref}, moderationc.ActionApprove, 0); err != nil {
				return fmt.Errorf("seed approve: %w", err)
			}
			decided["approved, the filter held it wrongly"]++
		case x.status == "pending" && toxic:
			if _, err := modSvc.Decide(managerCtx, []contentc.Ref{x.ref}, moderationc.ActionReject, x.rule); err != nil {
				return fmt.Errorf("seed reject: %w", err)
			}
			decided["rejected"]++
		case x.status == "published" && toxic:
			if err := modSvc.Remove(managerCtx, x.ref, x.rule); err != nil {
				return fmt.Errorf("seed remove: %w", err)
			}
			decided["removed, the filter missed it"]++
		}
	}
	for k, n := range decided {
		log.Info("seed: manager pass", slog.String("outcome", k), slog.Int("items", n))
	}

	// 4. Minh yêu cầu xem xét lại một quyết định; quản trị viên (người khác) sẽ xử lý trong app
	minhCtx, err := as("minh")
	if err != nil {
		return err
	}
	mine, err := modSvc.MyDecisions(minhCtx)
	if err != nil {
		return err
	}
	for _, md := range mine {
		if md.CanAppeal {
			if _, err := modSvc.CreateAppeal(minhCtx, md.ID,
				"Mình chỉ bức xúc vì xe máy bị chắn lối mỗi tối, không nhằm vào cá nhân nào. Mình sẽ nói nhẹ nhàng hơn."); err != nil {
				return err
			}
			log.Info("seed: one appeal is waiting for the admin")
			break
		}
	}
	log.Info("seed: demo data created; the worker delivers notifications and strikes", slog.Int("accounts", len(people)))
	return nil
}

// waitFor gọi check mỗi giây cho tới khi nó trả true hoặc hết thời gian
func waitFor(ctx context.Context, limit time.Duration, check func() (bool, error)) (bool, error) {
	deadline := time.Now().Add(limit)
	for {
		ok, err := check()
		if err != nil || ok {
			return ok, err
		}
		if time.Now().After(deadline) {
			return false, nil
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// --- Kế hoạch nội dung ---

// Ý định do người viết seed gán; mô hình không biết điều này, nó tự chấm
const (
	intentClean     = "clean"
	intentOffensive = "offensive"
	intentHate      = "hate"
	intentKeyword   = "keyword"
	intentForeign   = "foreign"
)

// plan: ý định, quy tắc khi bị xử lý, có để lại trong hàng chờ không, và lùi thời gian bao lâu
type plan struct {
	intent string
	rule   int
	leave  bool
	ago    time.Duration
}

type created struct {
	ref    contentc.Ref
	status string
	plan
}

type postSpec struct {
	key, who, channel, title, body string
	pin                            bool
	plan
}

type commentSpec struct {
	key, who, post, parent, body string
	plan
}

const (
	hour = time.Hour
	day  = 24 * time.Hour
)

func clean(ago time.Duration) plan { return plan{intent: intentClean, ago: ago} }

func bad(intent string, rule int, ago time.Duration) plan {
	return plan{intent: intent, rule: rule, ago: ago}
}

// leaveInQueue: để mục này lại trong hàng chờ cho người dùng tự xử lý
func leaveInQueue(p plan) plan {
	p.leave = true
	return p
}

var posts = []postSpec{
	// Thông báo chung: ban quản lý
	{key: "water", who: "manager", channel: "thong-bao", pin: true, title: "Lịch cắt nước bảo trì bể chứa tháng 11",
		body: "Ban quản lý thông báo: tòa A và B sẽ tạm ngưng cấp nước từ 8:00 đến 12:00 sáng thứ Bảy để vệ sinh bể chứa.\n\nCư dân vui lòng trữ nước trước. Xin cảm ơn sự hợp tác của cư dân.",
		plan: clean(6 * day)},
	{key: "lift", who: "manager", channel: "thong-bao", title: "Bảo trì thang máy tòa B vào thứ Ba",
		body: "Thang máy số 2 tòa B sẽ dừng hoạt động từ 9:00 đến 15:00 thứ Ba để thay cáp. Thang số 1 vẫn chạy bình thường. Mong cư dân thông cảm.",
		plan: clean(3 * day)},
	{key: "waste", who: "manager", channel: "thong-bao", title: "Phân loại rác tại nguồn từ ngày 1/12",
		body: "Từ ngày 1/12, mỗi tầng có ba thùng: rác hữu cơ, rác tái chế và rác còn lại. Ban quản lý sẽ phát túi phân loại miễn phí tại sảnh trong tuần này.",
		plan: clean(30 * hour)},
	{key: "fire", who: "manager", channel: "thong-bao", title: "Diễn tập phòng cháy chữa cháy sáng Chủ nhật",
		body: "Buổi diễn tập bắt đầu lúc 8:30 sáng Chủ nhật. Chuông báo cháy sẽ kêu khoảng 10 phút. Cư dân không cần sơ tán nếu không muốn tham gia.",
		plan: clean(8 * hour)},

	// Thảo luận chung
	{key: "parking", who: "hoa", channel: "thao-luan", title: "Gửi xe máy ở hầm B2 có ai thấy chật không?",
		body: "Dạo này buổi tối về muộn gần như không còn chỗ ở hầm B2. Mọi người có đề xuất gì để sắp xếp lại không?",
		plan: clean(4 * day)},
	{key: "kids", who: "thao", channel: "thao-luan", title: "Trẻ con chơi ở sảnh buổi trưa",
		body: "Mấy đứa trẻ chơi ở sảnh ồn ào quá, nhất là giờ nghỉ trưa. Phụ huynh nhắc giúp các bé với nhé.",
		plan: clean(2 * day)},
	{key: "slow", who: "tuan", channel: "thao-luan", title: "Cổng xe tòa C ba tuần chưa sửa",
		body: "Ban quản lý làm việc chậm quá, ba tuần chưa sửa xong cổng xe tòa C. Sáng nào cũng phải xuống mở tay.",
		plan: clean(3 * day)},
	{key: "election", who: "khoa", channel: "thao-luan", title: "Bầu trưởng ban đại diện cư dân",
		body: "Ngu như bò mà cũng đòi làm trưởng ban. Năm nay đừng ai bầu cho ông đó nữa.",
		plan: bad(intentHate, 1, 20*hour)},
	{key: "gym", who: "kevin", channel: "thao-luan", title: "Gym opening hours on Sunday?",
		body: "Does anyone know if the gym on the 2nd floor is open on Sunday mornings?",
		plan: plan{intent: intentForeign, ago: 26 * hour}},
	{key: "dog", who: "ngoc", channel: "thao-luan", title: "Chó sủa cả đêm ở tầng 9 tòa C",
		body: "Nhà nào để chó sủa cả đêm thì liệu hồn, đừng trách tôi ra tay.",
		plan: leaveInQueue(bad(intentHate, 3, 10*hour))},
	{key: "aircon", who: "ngoc", channel: "thao-luan", title: "Xin giới thiệu thợ sửa máy lạnh",
		body: "Ai có thợ sửa máy lạnh uy tín giới thiệu mình với, máy nhà mình chảy nước mấy hôm nay.",
		plan: clean(5 * hour)},

	// Bảo trì
	{key: "light", who: "minh", channel: "bao-tri", title: "Đèn hành lang tầng 8 tòa B bị chập chờn",
		body: "Đèn ở đoạn gần thang máy tầng 8 tòa B nhấp nháy mấy hôm nay, buổi tối đi lại hơi khó nhìn. Nhờ ban quản lý kiểm tra giúp.",
		plan: clean(5 * day)},
	{key: "liftbroken", who: "lan", channel: "bao-tri", title: "Thang máy tòa B lại dừng giữa tầng",
		body: "Thang máy tòa B lại hỏng rồi, sáng nay dừng giữa tầng 6 và 7 khoảng năm phút. Mong ban quản lý sửa sớm.",
		plan: clean(26 * hour)},
	{key: "gas", who: "thao", channel: "bao-tri", title: "Có mùi gas ở hành lang tầng 12",
		body: "Tầng 12 tòa A có mùi gas từ chiều, mọi người kiểm tra bếp giúp nhé. Mình đã báo bảo vệ.",
		plan: clean(3 * hour)},
	{key: "guard", who: "tuan", channel: "bao-tri", title: "Mất xe đạp ở hầm B1",
		body: "Bảo vệ ca đêm ngủ gật, mất cả xe đạp mà không ai hay. Nhờ ban quản lý trích camera giúp.",
		plan: clean(15 * hour)},

	// Marketplace
	{key: "fryer", who: "lan", channel: "marketplace", title: "Pass lại nồi chiên không dầu còn mới",
		body: "Nồi chiên 5 lít dùng được 3 tháng, còn bảo hành. Ai cần nhắn mình qua diễn đàn nhé.",
		plan: clean(2 * day)},
	{key: "loan", who: "minh", channel: "marketplace", title: "Hỗ trợ tài chính cho cư dân",
		body: "Vay nhanh trong ngày, lãi cực thấp, không cần thế chấp. Nhắn tin riêng để được tư vấn.",
		plan: bad(intentKeyword, 6, 36*hour)},
	{key: "bike", who: "kevin", channel: "marketplace", title: "Bán xe đạp trẻ em",
		body: "Xe đạp 16 inch cho bé 4 đến 6 tuổi, còn tốt, có bánh phụ. Giao nhận ở sảnh tòa B.",
		plan: clean(20 * hour)},
	{key: "books", who: "thao", channel: "marketplace", title: "Tặng sách giáo khoa lớp 5",
		body: "Nhà mình còn bộ sách giáo khoa lớp 5 năm ngoái, ai cần thì để lại bình luận, mình mang xuống sảnh.",
		plan: clean(9 * hour)},
}

var comments = []commentSpec{
	{key: "q1", who: "lan", post: "water", body: "Cảm ơn ban quản lý đã báo sớm. Thứ Bảy tuần này hay tuần sau ạ?", plan: clean(6*day - 2*hour)},
	{who: "manager", post: "water", parent: "q1", body: "Thứ Bảy tuần này chị nhé, ngày 21/11.", plan: clean(6*day - 3*hour)},
	{who: "khoa", post: "waste", body: "Phân loại rác phiền vl, ai rảnh mà làm.", plan: bad(intentOffensive, 4, 28*hour)},
	{who: "ngoc", post: "waste", body: "Ủng hộ ạ, nhà mình đã tách rác từ lâu rồi.", plan: clean(27 * hour)},
	{who: "tuan", post: "lift", body: "Thứ Ba mình phải chở mẹ đi khám, may còn thang số 1.", plan: clean(2 * day)},

	{key: "p1", who: "lan", post: "parking", body: "Mình cũng thấy vậy, chắc nên kẻ lại vạch cho xe máy.", plan: clean(4*day - 2*hour)},
	{who: "minh", post: "parking", body: "Ai để xe lấn chỗ thì đúng là đồ khốn, ngu không chịu được.", plan: bad(intentHate, 1, 4*day-3*hour)},
	{who: "tuan", post: "parking", parent: "p1", body: "Đề xuất kẻ vạch riêng cho xe máy điện, sạc ở góc trong.", plan: clean(4*day - 5*hour)},
	{who: "manager", post: "parking", body: "Ban quản lý sẽ kẻ lại vạch hầm B2 vào cuối tuần này. Cảm ơn cư dân đã góp ý.", plan: clean(3 * day)},

	{who: "ngoc", post: "kids", body: "Trẻ con mà, mình thấy cũng bình thường, chỉ cần tránh giờ trưa.", plan: clean(2*day - hour)},
	{who: "khoa", post: "kids", body: "Bọn thuê nhà toàn lũ rác rưởi, cấm cho thuê đi.", plan: bad(intentHate, 2, 2*day-2*hour)},

	{who: "manager", post: "slow", body: "Ban quản lý xin lỗi vì chậm trễ. Linh kiện cổng đã về, kỹ thuật sẽ lắp xong trong thứ Năm.", plan: clean(2 * day)},
	{who: "hoa", post: "slow", body: "Mong là lần này sửa dứt điểm, tuần trước cũng hứa rồi.", plan: clean(47 * hour)},

	{who: "manager", post: "light", body: "Đã ghi nhận, kỹ thuật sẽ thay bóng trong chiều nay.", plan: clean(5*day - 3*hour)},
	{who: "minh", post: "light", body: "Đã sáng lại rồi, cảm ơn kỹ thuật nhanh tay.", plan: clean(4 * day)},

	{who: "khoa", post: "liftbroken", body: "Cái thang máy chết tiệt này lại hỏng nữa rồi.", plan: clean(25 * hour)},
	{who: "tuan", post: "liftbroken", body: "Đm thang máy lại hỏng, nóng vl.", plan: bad(intentOffensive, 4, 24*hour)},
	{who: "manager", post: "liftbroken", body: "Kỹ thuật đã kiểm tra, lỗi do cảm biến cửa. Đã thay và chạy thử ổn định.", plan: clean(22 * hour)},

	{who: "manager", post: "gas", body: "Kỹ thuật đã lên kiểm tra, một van gas bị hở và đã được khóa. Cảm ơn chị đã báo kịp thời.", plan: clean(2 * hour)},
	{who: "lan", post: "gas", body: "Hú hồn, may mà phát hiện sớm.", plan: clean(90 * time.Minute)},

	{who: "khoa", post: "guard", body: "Óc chó thật sự, đổ rác ngay cửa thang máy mà cũng không ai nhắc.", plan: bad(intentHate, 1, 14*hour)},
	{who: "manager", post: "guard", body: "Ban quản lý đã trích camera và làm việc với đơn vị bảo vệ. Sẽ cập nhật cho anh trong hôm nay.", plan: clean(12 * hour)},

	{who: "hoa", post: "fryer", body: "Mình lấy nhé, tối nay qua sảnh A được không?", plan: clean(2*day - 4*hour)},
	{who: "tuan", post: "bike", body: "Ai cần tiền gấp thì vay nhanh bên mình, lãi cực thấp, nhắn mình nhé.", plan: leaveInQueue(bad(intentKeyword, 6, 19*hour))},
	{who: "ngoc", post: "books", body: "Cho mình xin bộ sách nhé, con mình vừa lên lớp 5.", plan: clean(8 * hour)},
}
