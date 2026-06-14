# Psiphon + V2Ray AAR (New API)

## Mục tiêu
Build `psiphon_v2ray.aar` mới bằng `gomobile bind`, gộp:
- **Xray-core** (V2Ray mới nhất) - API mirror libv2ray1.aar
- **Psiphon tunnel-core**

## Cấu trúc

```
psiphon_v2ray_aar/
├── go.mod
├── psiphon_v2ray/
│   ├── doc.go          - Package declaration
│   ├── v2ray.go        - V2Ray/Xray-core bridge
│   ├── psiphon.go      - Psiphon tunnel bridge
│   └── tun.go          - TUN fd + socket protect helper
├── android/
│   └── V2RayManager.java  - Java wrapper dùng AAR mới
└── .github/workflows/
    └── build.yml       - GitHub Actions CI build
```

## Java API sau khi build

Package: `psiphon_v2ray`

### V2Ray
```java
// Init (gọi 1 lần trong Application hoặc Service)
Psiphon_v2ray.initCoreEnv(assetPath, userPath);
Psiphon_v2ray.setSocketProtector(fd -> vpnService.protect(fd));

// Tạo controller
CoreController ctrl = Psiphon_v2ray.newCoreController(new CoreCallbackHandler() {
    public long onEmitStatus(long code, String status) { ... return 0; }
    public long shutdown() { ... return 0; }
    public long startup() { ... return 0; }
});

// Start (tunFd từ VpnService.Builder.establish())
ctrl.startLoop(configJSON, tunFd);

// Stop
ctrl.stopLoop();

// Stats
long bytes = ctrl.queryStats("proxy", "uplink");
```

### Psiphon
```java
PsiphonController psiphon = Psiphon_v2ray.newPsiphonController(new PsiphonNoticeHandler() {
    public void onNotice(String json, long ts, boolean diag) {
        // parse json để lấy socks port, tunnel count, v.v.
    }
});

psiphon.startTunnel(psiphonConfigJSON);
int socksPort = psiphon.getSocksPort(); // sau khi ListeningSocksProxyPort notice
psiphon.stopTunnel();
```

### SocketProtector
```java
// Bắt buộc phải set trước khi start tunnel
// Để Psiphon và V2Ray outbound không bị route qua VPN
Psiphon_v2ray.setSocketProtector(fd -> vpnService.protect(fd));
```

## Build bằng GitHub Actions

1. Push code lên GitHub
2. Actions tự chạy `.github/workflows/build.yml`
3. Download `psiphon_v2ray.aar` từ Artifacts

## Build thủ công (nếu có Linux)

```bash
# Setup
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init

# Build
gomobile bind \
  -target=android/arm,android/arm64 \
  -androidapi 21 \
  -o psiphon_v2ray.aar \
  github.com/caothemanh/psiphon_v2ray/psiphon_v2ray
```

## Khác biệt so với chzPsiphonAndV2ray.aar (cũ)

| | chzPsiphonAndV2ray.aar (cũ) | psiphon_v2ray.aar (mới) |
|---|---|---|
| V2Ray API | `V2RayPoint.runLoop()` | `CoreController.startLoop(config, tunFd)` |
| Psiphon API | `ChzPsiphonAndV2ray.start()` | `PsiphonController.startTunnel()` |
| Socket protect | `V2RayVPNServiceSupportsSet.protect()` | `SocketProtector.protectFd()` |
| Status callback | `V2RayVPNServiceSupportsSet.onEmitStatus()` | `CoreCallbackHandler.onEmitStatus()` |
| Psiphon notice | `PsiphonProvider.notice()` | `PsiphonNoticeHandler.onNotice()` |
| Xray version | cũ | mới nhất |

## Lưu ý quan trọng

### go.mod - cần update module path
Thay `github.com/yourname/psiphon_v2ray_aar` bằng GitHub repo thật của bạn.

### Psiphon credentials
Điền vào config JSON:
- `PropagationChannelId` - lấy từ APK Psiphon (đã reverse trước đây)
- `SponsorId` - lấy từ APK Psiphon

### Psiphon go module
`psiphon-tunnel-core` cần replace directive trong go.mod:
```
replace github.com/Psiphon-Labs/psiphon-tunnel-core => github.com/rod-hynes/psiphon-tunnel-core v0.0.0-<commit>
```
(Dùng fork rod-hynes nếu cần fix gomobile compatibility)
