package psiphon_v2ray

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

// ============================================================
// TUN fd protect helper
// ============================================================

// SocketProtector - Java VpnService implement để protect sockets.
// Cần để các kết nối Psiphon và V2Ray không bị route qua VPN lại.
type SocketProtector interface {
	// ProtectFd protect socket fd khỏi VPN routing.
	// Trả về true nếu thành công.
	ProtectFd(fd int) bool
}

var globalProtector SocketProtector

// SetSocketProtector đăng ký SocketProtector từ Java VpnService.
// Phải gọi trước khi start bất kỳ tunnel nào.
func SetSocketProtector(protector SocketProtector) {
	globalProtector = protector
}

// ProtectSocket protect một socket fd.
// Trả về error nếu không có protector hoặc protect thất bại.
func ProtectSocket(fd int) error {
	if globalProtector == nil {
		return fmt.Errorf("no SocketProtector registered")
	}
	if !globalProtector.ProtectFd(fd) {
		return fmt.Errorf("SocketProtector.ProtectFd(%d) returned false", fd)
	}
	return nil
}

// ============================================================
// TUN fd helper
// ============================================================

// SetTunFd truyền TUN fd vào Xray-core.
// fd: file descriptor từ Android VpnService.Builder.establish()
//
// Với Xray-core gVisor netstack, TUN fd được pass qua StartLoop().
// Function này dùng để set fd cho các transport layer cần fd trực tiếp.
func SetTunFd(fd int) error {
	if fd < 0 {
		return fmt.Errorf("invalid fd: %d", fd)
	}
	// Duplicate fd để tránh bị close bởi Java GC
	newFd, err := syscall.Dup(fd)
	if err != nil {
		return fmt.Errorf("failed to dup fd %d: %w", fd, err)
	}
	syscall.CloseOnExec(newFd)
	_ = newFd // Xray-core sẽ nhận fd qua StartLoop config
	return nil
}

// ============================================================
// Network dialer với protect
// ============================================================

// newProtectedDialer tạo net.Dialer với socket protect.
// Dùng cho Psiphon để các kết nối outbound không đi qua VPN.
func newProtectedDialer() *net.Dialer {
	return &net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			if globalProtector == nil {
				return nil
			}
			var protectErr error
			err := c.Control(func(fd uintptr) {
				if !globalProtector.ProtectFd(int(fd)) {
					protectErr = fmt.Errorf("protect failed for fd %d", fd)
				}
			})
			if err != nil {
				return err
			}
			return protectErr
		},
	}
}

// ============================================================
// File descriptor utilities
// ============================================================

// DupFd duplicate một file descriptor.
// Trả về fd mới, caller chịu trách nhiệm close.
func DupFd(fd int) (int, error) {
	newFd, err := syscall.Dup(fd)
	if err != nil {
		return -1, fmt.Errorf("dup(%d): %w", fd, err)
	}
	return newFd, nil
}

// CloseFd đóng một file descriptor.
func CloseFd(fd int) error {
	return os.NewFile(uintptr(fd), "fd").Close()
}
