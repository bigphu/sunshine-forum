package service

import (
	"sync"

	"github.com/google/uuid"
)

// Broker phân phát tín hiệu "có thông báo mới" tới các kết nối Server-Sent Events đang mở
// trong tiến trình API. Thông báo do worker ghi; DB phát NOTIFY và Listen gọi Publish.
type Broker struct {
	mu   sync.Mutex
	subs map[uuid.UUID]map[chan struct{}]struct{}
}

func NewBroker() *Broker { return &Broker{subs: map[uuid.UUID]map[chan struct{}]struct{}{}} }

// Subscribe trả kênh nhận tín hiệu và hàm huỷ đăng ký
func (b *Broker) Subscribe(account uuid.UUID) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	b.mu.Lock()
	if b.subs[account] == nil {
		b.subs[account] = map[chan struct{}]struct{}{}
	}
	b.subs[account][ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		delete(b.subs[account], ch)
		if len(b.subs[account]) == 0 {
			delete(b.subs, account)
		}
		b.mu.Unlock()
	}
}

// Publish không chặn: kênh đã có tín hiệu chưa đọc thì bỏ qua (client tải lại cả hộp thư)
func (b *Broker) Publish(account uuid.UUID) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs[account] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
