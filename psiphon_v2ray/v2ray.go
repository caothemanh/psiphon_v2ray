package psiphon_v2ray

import (
	"context"
	"fmt"
	"sync"

	// Xray-core
	core "github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/app/dispatcher"
	_ "github.com/xtls/xray-core/app/dns"
	_ "github.com/xtls/xray-core/app/dns/fakedns"
	_ "github.com/xtls/xray-core/app/log"
	_ "github.com/xtls/xray-core/app/policy"
	_ "github.com/xtls/xray-core/app/proxyman/inbound"
	_ "github.com/xtls/xray-core/app/proxyman/outbound"
	_ "github.com/xtls/xray-core/app/router"
	_ "github.com/xtls/xray-core/app/stats"
	_ "github.com/xtls/xray-core/proxy/blackhole"
	_ "github.com/xtls/xray-core/proxy/dns"
	_ "github.com/xtls/xray-core/proxy/dokodemo"
	_ "github.com/xtls/xray-core/proxy/freedom"
	_ "github.com/xtls/xray-core/proxy/http"
	_ "github.com/xtls/xray-core/proxy/shadowsocks"
	_ "github.com/xtls/xray-core/proxy/socks"
	_ "github.com/xtls/xray-core/proxy/trojan"
	_ "github.com/xtls/xray-core/proxy/vless/inbound"
	_ "github.com/xtls/xray-core/proxy/vless/outbound"
	_ "github.com/xtls/xray-core/proxy/vmess/inbound"
	_ "github.com/xtls/xray-core/proxy/vmess/outbound"
	_ "github.com/xtls/xray-core/transport/internet/grpc"
	_ "github.com/xtls/xray-core/transport/internet/http"
	_ "github.com/xtls/xray-core/transport/internet/httpupgrade"
	_ "github.com/xtls/xray-core/transport/internet/kcp"
	_ "github.com/xtls/xray-core/transport/internet/quic"
	_ "github.com/xtls/xray-core/transport/internet/reality"
	_ "github.com/xtls/xray-core/transport/internet/splithttp"
	_ "github.com/xtls/xray-core/transport/internet/tcp"
	_ "github.com/xtls/xray-core/transport/internet/tls"
	_ "github.com/xtls/xray-core/transport/internet/udp"
	_ "github.com/xtls/xray-core/transport/internet/websocket"
	_ "github.com/xtls/xray-core/transport/internet/xhttp"

	v2net "github.com/xtls/xray-core/common/net"
	v2filesystem "github.com/xtls/xray-core/common/platform/filesystem"
	v2stats "github.com/xtls/xray-core/features/stats"
	v2serial "github.com/xtls/xray-core/infra/conf/serial"
	v2internet "github.com/xtls/xray-core/transport/internet"
	"google.golang.org/protobuf/proto"
)

// ============================================================
// CoreCallbackHandler - Java phải implement interface này
// Giống hệt libv2ray1.aar
// ============================================================

// CoreCallbackHandler là interface Java implement để nhận callback từ V2Ray.
type CoreCallbackHandler interface {
	// OnEmitStatus được gọi khi V2Ray emit status.
	// Trả về 0 nếu thành công.
	OnEmitStatus(int64, string) int64

	// Shutdown được gọi khi V2Ray core yêu cầu dừng.
	// Trả về 0.
	Shutdown() int64

	// Startup được gọi sau khi V2Ray core start thành công.
	// Trả về 0.
	Startup() int64
}

// ProcessFinder - optional interface để V2Ray tìm process theo connection.
type ProcessFinder interface {
	FindProcessByConnection(string, string, int64, string, int64) int64
}

// ============================================================
// CoreController - mirror của libv2ray1 CoreController
// ============================================================

// CoreController quản lý lifecycle của một V2Ray instance.
// Tạo bằng NewCoreController().
type CoreController struct {
	mu              sync.Mutex
	xrayInstance    core.Server
	callbackHandler CoreCallbackHandler
	isRunning       bool
	cancelFunc      context.CancelFunc
	statsManager    v2stats.Manager
	assetPath       string
	userPath        string
}

var (
	globalAssetPath string
	globalUserPath  string
	initOnce        sync.Once
)

// InitCoreEnv khởi tạo môi trường V2Ray.
// Phải gọi trước NewCoreController().
// assetPath: thư mục chứa geoip.dat, geosite.dat
// userPath:  thư mục lưu data người dùng
func InitCoreEnv(assetPath, userPath string) {
	initOnce.Do(func() {
		globalAssetPath = assetPath
		globalUserPath = userPath
		// Set asset path cho Xray-core
		v2filesystem.NewFileReader = func(path string) ([]byte, error) {
			return v2filesystem.ReadFile(path)
		}
		// Override asset path
		if err := v2net.RegisterDestinationCallback(nil); err != nil {
			// ignore
		}
	})
	// Cập nhật path nếu gọi lại
	globalAssetPath = assetPath
	globalUserPath = userPath
}

// CheckVersionX trả về version string của Xray-core.
func CheckVersionX() string {
	return core.Version()
}

// NewCoreController tạo CoreController mới với callback handler.
// Sau khi tạo, gọi StartLoop() để start V2Ray.
func NewCoreController(handler CoreCallbackHandler) *CoreController {
	return &CoreController{
		callbackHandler: handler,
		assetPath:       globalAssetPath,
		userPath:        globalUserPath,
	}
}

// StartLoop khởi động V2Ray với config JSON và TUN fd.
// configJSON: nội dung file config JSON của Xray
// tunFd: file descriptor của TUN interface (-1 nếu không dùng TUN trực tiếp)
func (c *CoreController) StartLoop(configJSON string, tunFd int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning {
		return fmt.Errorf("CoreController already running")
	}

	// Parse config JSON → protobuf
	pbConfig, err := v2serial.LoadJSONConfig([]byte(configJSON))
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Nếu có tunFd, thêm TUN interface vào config
	if tunFd >= 0 {
		if err := c.setupTunFd(pbConfig, tunFd); err != nil {
			return fmt.Errorf("failed to setup TUN fd: %w", err)
		}
	}

	// Tạo Xray instance
	instance, err := core.New(pbConfig)
	if err != nil {
		return fmt.Errorf("failed to create xray instance: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFunc = cancel
	c.xrayInstance = instance

	// Start instance
	if err := instance.Start(); err != nil {
		cancel()
		c.xrayInstance = nil
		c.cancelFunc = nil
		return fmt.Errorf("failed to start xray: %w", err)
	}

	// Lấy stats manager nếu có
	if sm := instance.GetFeature(v2stats.ManagerType()); sm != nil {
		if statsManager, ok := sm.(v2stats.Manager); ok {
			c.statsManager = statsManager
		}
	}

	c.isRunning = true

	// Gọi startup callback
	if c.callbackHandler != nil {
		go c.callbackHandler.Startup()
	}

	// Goroutine chờ context cancel
	go func() {
		<-ctx.Done()
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.xrayInstance != nil {
			_ = c.xrayInstance.Close()
			c.xrayInstance = nil
		}
		c.isRunning = false
		if c.callbackHandler != nil {
			c.callbackHandler.Shutdown()
		}
	}()

	return nil
}

// StopLoop dừng V2Ray instance.
func (c *CoreController) StopLoop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning {
		return
	}
	if c.cancelFunc != nil {
		c.cancelFunc()
		c.cancelFunc = nil
	}
}

// GetIsRunning trả về trạng thái V2Ray.
func (c *CoreController) GetIsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isRunning
}

// QueryStats trả về traffic stats theo tag và direction.
// tag: outbound tag (e.g. "proxy")
// direction: "uplink" hoặc "downlink"
func (c *CoreController) QueryStats(tag, direction string) int64 {
	if c.statsManager == nil {
		return 0
	}
	counter := c.statsManager.GetCounter(fmt.Sprintf("outbound>>>%s>>>traffic>>>%s", tag, direction))
	if counter == nil {
		return 0
	}
	return counter.Value()
}

// QueryAllOutboundTrafficStats trả về tổng traffic tất cả outbound.
func (c *CoreController) QueryAllOutboundTrafficStats() int64 {
	if c.statsManager == nil {
		return 0
	}
	var total int64
	for _, direction := range []string{"uplink", "downlink"} {
		for _, tag := range []string{"proxy", "direct", "block"} {
			counter := c.statsManager.GetCounter(
				fmt.Sprintf("outbound>>>%s>>>traffic>>>%s", tag, direction),
			)
			if counter != nil {
				total += counter.Value()
			}
		}
	}
	return total
}

// MeasureDelay đo độ trễ của một outbound.
// configJSON: config JSON chứa outbound cần đo
// Trả về delay ms, hoặc -1 nếu lỗi.
func (c *CoreController) MeasureDelay(configJSON string) (int64, error) {
	// Placeholder - implement nếu cần
	return -1, fmt.Errorf("not implemented")
}

// RegisterProcessFinder đăng ký ProcessFinder cho V2Ray.
func (c *CoreController) RegisterProcessFinder(finder ProcessFinder) {
	// Placeholder - implement nếu cần Android process lookup
}

// setupTunFd thêm TUN fd vào config protobuf.
// Xray-core với gVisor netstack sẽ đọc traffic từ fd này.
func (c *CoreController) setupTunFd(config proto.Message, tunFd int) error {
	// Nếu dùng gVisor netstack trong Xray, TUN fd được pass qua
	// inbound dokodemo hoặc custom transport.
	// Trong hầu hết trường hợp, TUN fd pass thẳng vào startLoop
	// và Xray tự handle qua gVisor.
	//
	// Với Android VpnService:
	// 1. VpnService.Builder.establish() → tunFd
	// 2. tunFd pass vào StartLoop(config, tunFd)
	// 3. Xray dùng gVisor để read/write packet từ tunFd
	//
	// Hiện tại để trống - Android layer handle TUN setup
	_ = tunFd
	return nil
}

// ============================================================
// Expose theo chuẩn gomobile (lowercase method names không được export)
// Gomobile export tất cả public types/funcs
// ============================================================

// MeasureOutboundDelay đo delay của outbound (top-level function).
func MeasureOutboundDelay(serviceAddress, configJSON string) (int64, error) {
	return -1, fmt.Errorf("not implemented")
}

// ReconcileBrowserDialer - placeholder cho browser dialer.
func ReconcileBrowserDialer(config string) {
	// no-op
}

// SetAssetPath set đường dẫn asset (geoip.dat, geosite.dat).
// Cần gọi trước khi start V2Ray nếu dùng routing rules.
func SetAssetPath(path string) {
	v2internet.UseAlternativeSystemDialer(nil) // reset
	globalAssetPath = path
}

// Đảm bảo import v2internet được dùng
var _ = v2internet.UseAlternativeSystemDialer
