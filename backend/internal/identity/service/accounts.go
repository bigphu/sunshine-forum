package service

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"time"

	"github.com/bigphu/goat/auth"
	"github.com/bigphu/goat/errs"
	"github.com/google/uuid"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"sunshine/internal/identity/contract"
	"sunshine/internal/identity/domain"
)

// AccountView là một dòng trong danh sách tài khoản của Admin
type AccountView struct {
	domain.Account
	InviteExpiresAt *time.Time
}

// ListAccounts sắp theo vai trò (ban quản lý trước) rồi theo tên tiếng Việt
func (s *Service) ListAccounts(ctx context.Context) ([]AccountView, error) {
	if _, err := auth.Require(ctx, domain.PermAccountManage); err != nil {
		return nil, err
	}
	list, err := s.accounts.List(ctx)
	if err != nil {
		return nil, err
	}
	rank := map[domain.Role]int{domain.RoleAdmin: 0, domain.RoleManager: 1, domain.RoleResident: 2}
	col := collate.New(language.Vietnamese)
	slices.SortFunc(list, func(a, b domain.Account) int {
		return cmp.Or(cmp.Compare(rank[a.Role], rank[b.Role]), col.CompareString(a.Name, b.Name))
	})
	out := make([]AccountView, len(list))
	for i, a := range list {
		out[i] = AccountView{Account: a}
		if a.Status == domain.StatusInvited {
			if out[i].InviteExpiresAt, err = s.tokens.PendingInviteExpiry(ctx, a.ID); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

type Invitation struct {
	Email string
	Name  string
	Unit  string
	Role  domain.Role
}

// Invite thêm người từ danh sách cư dân và gửi thư mời đặt mật khẩu (FR-1.1).
// Không có đăng ký công khai.
func (s *Service) Invite(ctx context.Context, in Invitation) (AccountView, error) {
	if _, err := auth.Require(ctx, domain.PermAccountManage); err != nil {
		return AccountView{}, err
	}
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return AccountView{}, err
	}
	name, err := domain.CleanName(in.Name)
	if err != nil {
		return AccountView{}, err
	}
	unit, err := domain.CleanUnit(in.Unit)
	if err != nil {
		return AccountView{}, err
	}
	if !in.Role.Valid() {
		return AccountView{}, domain.ErrInvalidRole
	}
	raw, hash := newToken()
	expires := s.now().Add(s.cfg.InviteTTL)
	a, err := s.accounts.Create(ctx, domain.NewAccount{
		Email: email, Name: name, Unit: unit, Role: in.Role,
		Invite: domain.IssuedToken{Raw: raw, Hash: hash, Purpose: domain.PurposeInvite, ExpiresAt: expires},
	})
	if err != nil {
		return AccountView{}, err
	}
	return AccountView{Account: a, InviteExpiresAt: &expires}, nil
}

// ImportSkip là một dòng không mời được
type ImportSkip struct {
	Row    int
	Email  string
	Reason string
}

// Import mời một loạt cư dân từ danh sách đang thuê (lúc mới triển khai). Dòng lỗi được báo
// lại, không làm hỏng cả lô.
func (s *Service) Import(ctx context.Context, people []Invitation) (int, []ImportSkip, error) {
	if _, err := auth.Require(ctx, domain.PermAccountManage); err != nil {
		return 0, nil, err
	}
	invited := 0
	skipped := []ImportSkip{}
	for i, p := range people {
		p.Role = domain.RoleResident
		if _, err := s.Invite(ctx, p); err != nil {
			reason := err.Error()
			var e *errs.Error
			if errors.As(err, &e) {
				reason = e.Detail()
			}
			skipped = append(skipped, ImportSkip{Row: i + 1, Email: p.Email, Reason: reason})
			continue
		}
		invited++
	}
	return invited, skipped, nil
}

// Reinvite gửi link mời mới; link cũ hết hiệu lực
func (s *Service) Reinvite(ctx context.Context, id uuid.UUID) (AccountView, error) {
	if _, err := auth.Require(ctx, domain.PermAccountManage); err != nil {
		return AccountView{}, err
	}
	a, err := s.accounts.Get(ctx, id)
	if err != nil {
		return AccountView{}, err
	}
	if a.Status != domain.StatusInvited {
		return AccountView{}, domain.ErrNotInvited
	}
	raw, hash := newToken()
	expires := s.now().Add(s.cfg.InviteTTL)
	if err := s.tokens.Issue(ctx, id, domain.IssuedToken{Raw: raw, Hash: hash, Purpose: domain.PurposeInvite, ExpiresAt: expires}); err != nil {
		return AccountView{}, err
	}
	return AccountView{Account: a, InviteExpiresAt: &expires}, nil
}

// Deactivate khoá tài khoản khi hợp đồng thuê kết thúc; người đó bị đăng xuất ngay
func (s *Service) Deactivate(ctx context.Context, id uuid.UUID) (AccountView, error) {
	actor, err := auth.Require(ctx, domain.PermAccountManage)
	if err != nil {
		return AccountView{}, err
	}
	if actor.AccountID == id {
		return AccountView{}, domain.ErrSelfChange
	}
	a, err := s.accounts.Deactivate(ctx, id)
	return AccountView{Account: a}, err
}

func (s *Service) SetRole(ctx context.Context, id uuid.UUID, role domain.Role) (AccountView, error) {
	actor, err := auth.Require(ctx, domain.PermAccountManage)
	if err != nil {
		return AccountView{}, err
	}
	if actor.AccountID == id {
		return AccountView{}, domain.ErrSelfChange
	}
	if !role.Valid() {
		return AccountView{}, domain.ErrInvalidRole
	}
	a, err := s.accounts.SetRole(ctx, id, role)
	return AccountView{Account: a}, err
}

// --- contract.AccountReader ---

type accountReader struct{ s *Service }

func (s *Service) AccountReader() contract.AccountReader { return accountReader{s} }

func (r accountReader) GetAccounts(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]contract.Account, error) {
	list, err := r.s.accounts.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]contract.Account, len(list))
	for _, a := range list {
		out[a.ID] = contract.Account{ID: a.ID, Name: a.Name, Unit: a.Unit, Role: string(a.Role), Staff: a.Role.IsStaff()}
	}
	return out, nil
}

func (r accountReader) ActiveResidentIDs(ctx context.Context) ([]uuid.UUID, error) {
	return r.s.accounts.ActiveIDsByRole(ctx, []domain.Role{domain.RoleResident})
}
