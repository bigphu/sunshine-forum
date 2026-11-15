// Package service là use case của moderation: hàng chờ, duyệt / từ chối (cả hàng loạt),
// gỡ nội dung đã đăng, yêu cầu xem xét lại và nhật ký kiểm toán.
package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bigphu/goat/auth"
	"github.com/google/uuid"

	content "sunshine/internal/content/contract"
	identity "sunshine/internal/identity/contract"
	"sunshine/internal/moderation/contract"
	"sunshine/internal/moderation/domain"
	penalties "sunshine/internal/penalties/contract"
	"sunshine/internal/privacy"
	screening "sunshine/internal/screening/contract"
)

type Deps struct {
	Repo     domain.Repository
	Content  content.Reader
	Accounts identity.AccountReader
	Standing penalties.StandingReader
	Cipher   *privacy.Cipher
	Now      func() time.Time
}

type Service struct {
	repo     domain.Repository
	content  content.Reader
	accounts identity.AccountReader
	standing penalties.StandingReader
	cipher   *privacy.Cipher
	now      func() time.Time
}

func New(d Deps) *Service {
	now := d.Now
	if now == nil {
		now = time.Now
	}
	return &Service{repo: d.Repo, content: d.Content, accounts: d.Accounts, standing: d.Standing, cipher: d.Cipher, now: now}
}

// --- hàng chờ ---

type QueueItem struct {
	Case   domain.Case
	Item   content.Item
	Author identity.Account
	// AuthorStrikes: vi phạm đã xác nhận của người viết trong khoảng thời gian hiện tại
	AuthorStrikes int
}

type AppealView struct {
	Appeal        domain.Appeal
	Decision      domain.Decision
	Item          content.Item
	Author        identity.Account
	AuthorStrikes int
	Moderator     *identity.Account
	// CanReview: false khi người xem là người đã ra quyết định ban đầu
	CanReview bool
}

type Queue struct {
	Items   []QueueItem
	Appeals []AppealView
}

// Queue: nội dung chờ duyệt (cũ nhất trước) và yêu cầu xem xét lại (hàng ưu tiên, FR-4.4)
func (s *Service) Queue(ctx context.Context) (Queue, error) {
	actor, err := auth.Require(ctx, domain.PermReview)
	if err != nil {
		return Queue{}, err
	}
	cases, err := s.repo.OpenCases(ctx)
	if err != nil {
		return Queue{}, err
	}
	var appeals []domain.Appeal
	if actor.Can(domain.PermAppealReview) {
		if appeals, err = s.repo.PendingAppeals(ctx); err != nil {
			return Queue{}, err
		}
	}
	decisionIDs := make([]uuid.UUID, len(appeals))
	for i, a := range appeals {
		decisionIDs[i] = a.DecisionID
	}
	decisions, err := s.repo.Decisions(ctx, decisionIDs)
	if err != nil {
		return Queue{}, err
	}
	refs := make([]content.Ref, 0, len(cases)+len(appeals))
	people := []uuid.UUID{}
	for _, c := range cases {
		refs = append(refs, c.Ref)
		people = append(people, c.AuthorID)
	}
	for _, d := range decisions {
		refs = append(refs, d.Ref)
		people = append(people, d.AuthorID)
		if d.ModeratorID != nil {
			people = append(people, *d.ModeratorID)
		}
	}
	items, err := s.content.Items(ctx, refs)
	if err != nil {
		return Queue{}, err
	}
	accounts, err := s.accounts.GetAccounts(ctx, people)
	if err != nil {
		return Queue{}, err
	}
	strikes := map[uuid.UUID]int{}
	countFor := func(id uuid.UUID) (int, error) {
		if n, ok := strikes[id]; ok {
			return n, nil
		}
		st, err := s.standing.Standing(ctx, id)
		strikes[id] = st.ActiveStrikes
		return st.ActiveStrikes, err
	}
	var q Queue
	for _, c := range cases {
		it, ok := items[c.Ref]
		if !ok {
			continue
		}
		n, err := countFor(c.AuthorID)
		if err != nil {
			return Queue{}, err
		}
		q.Items = append(q.Items, QueueItem{Case: c, Item: it, Author: accounts[c.AuthorID], AuthorStrikes: n})
	}
	for _, a := range appeals {
		d := decisions[a.DecisionID]
		it, ok := items[d.Ref]
		if !ok {
			continue // nội dung đã bị xoá hẳn
		}
		n, err := countFor(d.AuthorID)
		if err != nil {
			return Queue{}, err
		}
		v := AppealView{Appeal: a, Decision: d, Item: it, Author: accounts[d.AuthorID], AuthorStrikes: n,
			CanReview: d.ModeratorID == nil || *d.ModeratorID != actor.AccountID}
		if d.ModeratorID != nil {
			m := accounts[*d.ModeratorID]
			v.Moderator = &m
		}
		q.Appeals = append(q.Appeals, v)
	}
	return q, nil
}

// --- quyết định ---

func (s *Service) deadline() *time.Time {
	t := s.now().Add(domain.AppealWindow)
	return &t
}

func (s *Service) audit(action string, moderator *uuid.UUID, author uuid.UUID, ref content.Ref, ai domain.AI, rule int, outcome string, overturned bool) domain.AuditEntry {
	e := domain.AuditEntry{
		Action: action, ModeratorID: moderator, Subject: s.cipher.Pseudonym(author), ContentType: ref.Type,
		AI: ai, Outcome: outcome, OverturnedAI: overturned,
	}
	if rule > 0 {
		e.Rule = &rule
	}
	return e
}

// Decide duyệt hoặc từ chối một hay nhiều nội dung đang chờ (FR-4.1, FR-4.3). Từ chối
// luôn phải nêu quy tắc, kể cả hàng loạt (FR-4.5). Mục người khác vừa xử lý bị bỏ qua.
func (s *Service) Decide(ctx context.Context, refs []content.Ref, action string, rule int) (int, error) {
	actor, err := auth.Require(ctx, domain.PermReview)
	if err != nil {
		return 0, err
	}
	if len(refs) == 0 {
		return 0, domain.ErrNothingSelected
	}
	if action != contract.ActionApprove && action != contract.ActionReject {
		return 0, domain.ErrNothingSelected
	}
	if action == contract.ActionReject && !screening.ValidRule(rule) {
		return 0, domain.ErrRuleRequired
	}
	if action == contract.ActionApprove {
		rule = 0
	}
	items, err := s.content.Items(ctx, refs)
	if err != nil {
		return 0, err
	}
	me := actor.AccountID
	var batch []domain.NewDecision
	for _, ref := range refs {
		it, ok := items[ref]
		if !ok || it.Status != content.StatusPending {
			continue
		}
		ai := domain.AIFrom(it.Screening)
		d := domain.Decision{Ref: ref, AuthorID: it.AuthorID, ChannelID: it.ChannelID, Action: action, Rule: rule,
			ModeratorID: &me, AI: ai}
		outcome := "published"
		if action == contract.ActionApprove {
			// Bộ lọc giữ nhầm: duyệt là đảo kết luận của AI
			d.OverturnedAI = ai.Flagged()
		} else {
			outcome = "rejected"
			d.AppealDeadline = s.deadline()
		}
		batch = append(batch, domain.NewDecision{
			Decision: d, CloseCase: true,
			Audit: s.audit(action, &me, it.AuthorID, ref, ai, rule, outcome, d.OverturnedAI),
		})
	}
	if len(batch) == 0 {
		return 0, nil
	}
	return s.repo.Record(ctx, batch, true)
}

// Remove gỡ nội dung bộ lọc đã cho đăng (FR-4.1, UC-19). Tính là vi phạm đã xác nhận,
// được xem xét lại như mọi lần từ chối, và ghi là một lần bộ lọc bỏ sót (FR-5.3).
func (s *Service) Remove(ctx context.Context, ref content.Ref, rule int) error {
	actor, err := auth.Require(ctx, domain.PermRemove)
	if err != nil {
		return err
	}
	if !screening.ValidRule(rule) {
		return domain.ErrRuleRequired
	}
	items, err := s.content.Items(ctx, []content.Ref{ref})
	if err != nil {
		return err
	}
	it, ok := items[ref]
	if !ok {
		return domain.ErrContentNotFound
	}
	if it.Status != content.StatusPublished {
		return domain.ErrNotPublished
	}
	me := actor.AccountID
	ai := domain.AIFrom(it.Screening)
	// Mô hình đã chấm và cho đăng: gỡ là đảo kết luận của AI
	overturned := ai.Reason == screening.ReasonModel
	_, err = s.repo.Record(ctx, []domain.NewDecision{{
		Decision: domain.Decision{Ref: ref, AuthorID: it.AuthorID, ChannelID: it.ChannelID, Action: contract.ActionRemove,
			Rule: rule, ModeratorID: &me, AI: ai, OverturnedAI: overturned, AppealDeadline: s.deadline()},
		Audit: s.audit(contract.ActionRemove, &me, it.AuthorID, ref, ai, rule, "removed", overturned),
	}}, false)
	return err
}

// --- subscriber ---

// OnHeld mở case khi content giữ nội dung lại (idempotent)
func (s *Service) OnHeld(ctx context.Context, e content.Held) error {
	return s.repo.OpenCase(ctx, e.Ref, e.AuthorID, e.ChannelID)
}

// OnGone đóng case mở khi nội dung được đăng (sửa thành sạch, quét lại) hoặc bị tác giả xoá
func (s *Service) OnGone(ctx context.Context, ref content.Ref) error {
	return s.repo.CloseCase(ctx, ref, domain.CaseWithdrawn)
}

// OnAutoDeleted ghi quyết định của bộ lọc (chỉ xảy ra khi Admin đã bật tự xoá)
func (s *Service) OnAutoDeleted(ctx context.Context, e content.AutoDeleted) error {
	items, err := s.content.Items(ctx, []content.Ref{e.Ref})
	if err != nil {
		return err
	}
	ai := domain.AI{}
	if it, ok := items[e.Ref]; ok {
		ai = domain.AIFrom(it.Screening)
	}
	_, err = s.repo.Record(ctx, []domain.NewDecision{{
		Decision: domain.Decision{Ref: e.Ref, AuthorID: e.AuthorID, ChannelID: e.ChannelID, Action: contract.ActionAutoDelete,
			Rule: e.Rule, AI: ai, AppealDeadline: s.deadline()},
		Audit: s.audit(contract.ActionAutoDelete, nil, e.AuthorID, e.Ref, ai, e.Rule, "auto_deleted", false),
	}}, false)
	return err
}

// --- xem xét lại ---

func (s *Service) CreateAppeal(ctx context.Context, decisionID uuid.UUID, reason string) (domain.Appeal, error) {
	actor, err := auth.Require(ctx, domain.PermAppealCreate)
	if err != nil {
		return domain.Appeal{}, err
	}
	d, err := s.repo.Decision(ctx, decisionID)
	if err != nil {
		return domain.Appeal{}, err
	}
	if d.AuthorID != actor.AccountID {
		return domain.Appeal{}, domain.ErrDecisionNotFound
	}
	if !d.Appealable(s.now()) {
		return domain.Appeal{}, domain.ErrNotAppealable
	}
	r, err := domain.CleanAppealReason(reason)
	if err != nil {
		return domain.Appeal{}, err
	}
	return s.repo.CreateAppeal(ctx, domain.Appeal{DecisionID: decisionID, AuthorID: actor.AccountID, Reason: r})
}

// ReviewAppeal: một quản lý khác hoặc Quản trị viên quyết định; kết quả là cuối cùng.
// Chấp nhận thì khôi phục nội dung và thu hồi vi phạm (qua event).
func (s *Service) ReviewAppeal(ctx context.Context, appealID uuid.UUID, accept bool) error {
	actor, err := auth.Require(ctx, domain.PermAppealReview)
	if err != nil {
		return err
	}
	me := actor.AccountID
	_, _, err = s.repo.DecideAppeal(ctx, appealID, accept, me, func(d domain.Decision) domain.AuditEntry {
		action, outcome := "appeal_declined", "permanently_rejected"
		overturned := false
		if accept {
			action, outcome = "appeal_accepted", "restored"
			overturned = d.Action == contract.ActionAutoDelete
		}
		return s.audit(action, &me, d.AuthorID, d.Ref, d.AI, d.Rule, outcome, overturned)
	})
	return err
}

// MyDecision là một quyết định về nội dung của người đang đăng nhập
type MyDecision struct {
	domain.DecisionWithAppeal
	Item      *content.Item
	CanAppeal bool
}

func (s *Service) MyDecisions(ctx context.Context) ([]MyDecision, error) {
	actor, err := auth.Require(ctx, domain.PermAppealCreate)
	if err != nil {
		return nil, err
	}
	list, err := s.repo.DecisionsByAuthor(ctx, actor.AccountID)
	if err != nil {
		return nil, err
	}
	refs := make([]content.Ref, len(list))
	for i, d := range list {
		refs[i] = d.Ref
	}
	items, err := s.content.Items(ctx, refs)
	if err != nil {
		return nil, err
	}
	now := s.now()
	out := make([]MyDecision, len(list))
	for i, d := range list {
		out[i] = MyDecision{DecisionWithAppeal: d, CanAppeal: d.Appeal == nil && d.Appealable(now)}
		if it, ok := items[d.Ref]; ok {
			out[i].Item = &it
		}
	}
	return out, nil
}

// --- nhật ký kiểm toán ---

type AuditView struct {
	domain.AuditEntry
	Moderator *identity.Account
}

func (s *Service) Audit(ctx context.Context, beforeID *int64, filter string) ([]AuditView, *int64, error) {
	if _, err := auth.Require(ctx, domain.PermAuditRead); err != nil {
		return nil, nil, err
	}
	const page = 50
	list, err := s.repo.Audit(ctx, beforeID, filter, page+1)
	if err != nil {
		return nil, nil, err
	}
	var next *int64
	if len(list) > page {
		list = list[:page]
		next = &list[page-1].ID
	}
	ids := []uuid.UUID{}
	for _, e := range list {
		if e.ModeratorID != nil && !slices.Contains(ids, *e.ModeratorID) {
			ids = append(ids, *e.ModeratorID)
		}
	}
	mods, err := s.accounts.GetAccounts(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	out := make([]AuditView, len(list))
	for i, e := range list {
		out[i] = AuditView{AuditEntry: e}
		if e.ModeratorID != nil {
			m := mods[*e.ModeratorID]
			out[i].Moderator = &m
		}
	}
	return out, next, nil
}

// --- contract.Reader ---

type reader struct{ s *Service }

func (s *Service) Reader() contract.Reader { return reader{s} }

func (r reader) Stats(ctx context.Context, since time.Time) (contract.Stats, error) {
	return r.s.repo.Stats(ctx, since)
}

// Corrections: quyết định đã chốt đảo kết luận của AI (bản xuất huấn luyện, US-6.4).
// Bỏ Quy tắc 5–7: mô hình không có nhãn cho chúng, và Quy tắc 5 chứa thông tin riêng tư.
func (r reader) Corrections(ctx context.Context, from, to time.Time) ([]contract.Correction, error) {
	list, err := r.s.repo.Corrections(ctx, from, to)
	if err != nil {
		return nil, err
	}
	var out []contract.Correction
	for _, d := range list {
		accepted := d.Appeal != nil && d.Appeal.Status == domain.AppealAccepted
		overturned := d.OverturnedAI || (accepted && d.Action == contract.ActionAutoDelete)
		if !overturned || d.Rule >= 5 {
			continue
		}
		label := "clean"
		if !accepted {
			switch {
			case d.Action == contract.ActionApprove:
				label = "clean"
			case d.Rule == 4:
				label = "offensive"
			case d.Rule >= 1 && d.Rule <= 3:
				label = "hate"
			}
		}
		out = append(out, contract.Correction{
			DecisionID: d.ID, ContentType: d.Ref.Type, ContentID: d.Ref.ID, Action: d.Action, FinalLabel: label,
			Rule: d.Rule, AIScore: d.AI.Score, ModelVersion: d.AI.ModelVersion, DecidedAt: d.CreatedAt,
		})
	}
	return out, nil
}

// --- đổi cấu hình (subscriber) ---

var configLabels = map[string]string{
	"hold_threshold":        "Giữ lại, ngưỡng chung",
	"auto_delete_threshold": "Tự động xóa, ngưỡng chung",
	"auto_delete_enabled":   "Tự động xóa",
	"outage_policy":         "Khi bộ lọc lỗi",
	"channel_hold":          "Giữ lại",
	"channel_auto_delete":   "Tự động xóa",
	"strike_limit":          "Số vi phạm",
	"strike_window_days":    "Số ngày tính vi phạm",
	"timeout_days":          "Số ngày chỉ đọc",
}

var configValues = map[string]string{
	"true": "bật", "false": "tắt", "fail_closed": "giữ lại", "fail_open": "vẫn đăng", "chung": "theo ngưỡng chung",
}

func configValue(v string) string {
	if x, ok := configValues[v]; ok {
		return x
	}
	return v
}

// ConfigChange là một giá trị cấu hình đã đổi, từ event của screening hay penalties
type ConfigChange struct {
	Field     string
	ChannelID *uuid.UUID
	Old, New  string
}

// OnConfigChanged ghi một dòng nhật ký cho mỗi lần lưu cấu hình (PRODUCT.md: thay đổi chính
// sách được ghi lại). Người thực hiện là actor của event.
func (s *Service) OnConfigChanged(ctx context.Context, eventID uuid.UUID, changes []ConfigChange) error {
	parts := make([]string, 0, len(changes))
	for _, c := range changes {
		label := configLabels[c.Field]
		if label == "" {
			label = c.Field
		}
		if c.ChannelID != nil {
			if ch, err := s.content.Channel(ctx, *c.ChannelID); err == nil {
				label += ", " + ch.Name
			}
		}
		parts = append(parts, fmt.Sprintf("%s: %s → %s", label, configValue(c.Old), configValue(c.New)))
	}
	if len(parts) == 0 {
		return nil
	}
	var actor *uuid.UUID
	if id := auth.ActorIDFrom(ctx); id != uuid.Nil {
		actor = &id
	}
	return s.repo.AuditConfig(ctx, eventID, actor, strings.Join(parts, "; "))
}
