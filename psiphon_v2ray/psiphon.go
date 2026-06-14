package psiphon_v2ray

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon"
)

// ============================================================
// PsiphonNoticeHandler - Java implement để nhận notice
// ============================================================

// PsiphonNoticeHandler nhận notice từ Psiphon tunnel core.
// Java phải implement interface này.
type PsiphonNoticeHandler interface {
	// OnNotice được gọi khi Psiphon emit notice JSON.
	// noticeJSON: chuỗi JSON notice từ Psiphon
	// timestamp:  epoch milliseconds
	// isDiagnostic: true nếu là diagnostic notice
	OnNotice(noticeJSON string, timestamp int64, isDiagnostic bool)
}

// ============================================================
// PsiphonController - quản lý Psiphon tunnel
// ============================================================

// PsiphonController quản lý lifecycle của Psiphon tunnel.
type PsiphonController struct {
	mu             sync.Mutex
	ctx            context.Context
	cancelFunc     context.CancelFunc
	noticeHandler  PsiphonNoticeHandler
	isRunning      bool
	socksPort      int
	httpPort       int
}

var psiphonOnce sync.Once

// NewPsiphonController tạo PsiphonController mới.
func NewPsiphonController(handler PsiphonNoticeHandler) *PsiphonController {
	return &PsiphonController{
		noticeHandler: handler,
	}
}

// StartTunnel khởi động Psiphon tunnel với config JSON.
//
// configJSON phải chứa các field:
//   - PropagationChannelId
//   - SponsorId
//   - DataRootDirectory
//   - LocalSocksProxyPort (0 = auto)
//
// Psiphon sẽ emit notice "ListeningSocksProxyPort" khi sẵn sàng.
// Lắng nghe qua PsiphonNoticeHandler.OnNotice().
func (p *PsiphonController) StartTunnel(configJSON string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return fmt.Errorf("Psiphon already running")
	}

	// Parse config để validate
	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &configMap); err != nil {
		return fmt.Errorf("invalid config JSON: %w", err)
	}

	// Thiết lập notice handler trước khi init
	psiphon.SetNoticeWriter(psiphon.NewNoticeReceiver(
		func(notice []byte) {
			if p.noticeHandler == nil {
				return
			}

			// Parse để lấy timestamp và type
			var n struct {
				Type      string `json:"noticeType"`
				Timestamp string `json:"timestamp"`
				Payload   json.RawMessage `json:"data"`
			}
			isDiagnostic := false
			timestamp := int64(0)

			if err := json.Unmarshal(notice, &n); err == nil {
				// Kiểm tra diagnostic notices
				diagnosticTypes := map[string]bool{
					"Info": true, "Alert": true, "Warning": true,
					"Debug": true, "InternalTunnelProtocol": true,
				}
				isDiagnostic = diagnosticTypes[n.Type]
			}

			p.noticeHandler.OnNotice(string(notice), timestamp, isDiagnostic)

			// Parse socks/http port từ notice
			p.parsePortNotice(notice)
		},
	))

	// Init Psiphon
	config, err := psiphon.LoadConfig([]byte(configJSON))
	if err != nil {
		return fmt.Errorf("failed to load psiphon config: %w", err)
	}

	if err := config.Commit(false); err != nil {
		return fmt.Errorf("failed to commit psiphon config: %w", err)
	}

	if err := psiphon.OpenDataStore(config); err != nil {
		return fmt.Errorf("failed to open psiphon data store: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.ctx = ctx
	p.cancelFunc = cancel
	p.isRunning = true

	// Start tunnel trong goroutine
	go func() {
		defer func() {
			p.mu.Lock()
			p.isRunning = false
			p.socksPort = 0
			p.httpPort = 0
			p.mu.Unlock()
		}()

		// Tạo tunnel controller
		tunnelController, err := psiphon.NewController(config)
		if err != nil {
			if p.noticeHandler != nil {
				errNotice := fmt.Sprintf(`{"noticeType":"Error","data":{"message":"%s"}}`, err.Error())
				p.noticeHandler.OnNotice(errNotice, 0, true)
			}
			return
		}

		// Run tunnel - blocking cho đến khi ctx cancel
		tunnelController.Run(ctx)
	}()

	return nil
}

// StopTunnel dừng Psiphon tunnel.
func (p *PsiphonController) StopTunnel() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return
	}

	if p.cancelFunc != nil {
		p.cancelFunc()
		p.cancelFunc = nil
	}

	psiphon.CloseDataStore()
}

// GetIsRunning trả về trạng thái Psiphon tunnel.
func (p *PsiphonController) GetIsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.isRunning
}

// GetSocksPort trả về SOCKS5 port hiện tại (0 nếu chưa sẵn sàng).
func (p *PsiphonController) GetSocksPort() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.socksPort
}

// GetHttpPort trả về HTTP proxy port hiện tại (0 nếu chưa sẵn sàng).
func (p *PsiphonController) GetHttpPort() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.httpPort
}

// parsePortNotice đọc port từ Psiphon notice JSON.
func (p *PsiphonController) parsePortNotice(notice []byte) {
	var n struct {
		Type string `json:"noticeType"`
		Data struct {
			Port int `json:"port"`
		} `json:"data"`
	}
	if err := json.Unmarshal(notice, &n); err != nil {
		return
	}
	switch n.Type {
	case "ListeningSocksProxyPort":
		p.mu.Lock()
		p.socksPort = n.Data.Port
		p.mu.Unlock()
	case "ListeningHttpProxyPort":
		p.mu.Lock()
		p.httpPort = n.Data.Port
		p.mu.Unlock()
	}
}

// ============================================================
// Top-level convenience functions (deprecated - dùng PsiphonController)
// ============================================================

var (
	globalPsiphonController *PsiphonController
	globalPsiphonMu         sync.Mutex
)

// StartPsiphon khởi động Psiphon với config JSON và notice handler.
// Convenience function - dùng NewPsiphonController() cho nhiều instance.
func StartPsiphon(configJSON string, handler PsiphonNoticeHandler) error {
	globalPsiphonMu.Lock()
	defer globalPsiphonMu.Unlock()

	if globalPsiphonController != nil && globalPsiphonController.GetIsRunning() {
		return fmt.Errorf("Psiphon already running, call StopPsiphon() first")
	}

	globalPsiphonController = NewPsiphonController(handler)
	return globalPsiphonController.StartTunnel(configJSON)
}

// StopPsiphon dừng Psiphon (dùng với StartPsiphon).
func StopPsiphon() {
	globalPsiphonMu.Lock()
	defer globalPsiphonMu.Unlock()

	if globalPsiphonController != nil {
		globalPsiphonController.StopTunnel()
		globalPsiphonController = nil
	}
}

// GetPsiphonSocksPort trả về SOCKS port của Psiphon đang chạy.
func GetPsiphonSocksPort() int {
	globalPsiphonMu.Lock()
	defer globalPsiphonMu.Unlock()
	if globalPsiphonController == nil {
		return 0
	}
	return globalPsiphonController.GetSocksPort()
}
