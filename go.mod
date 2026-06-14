module github.com/caothemanh/psiphon_v2ray

go 1.26

require (
    github.com/Psiphon-Labs/psiphon-tunnel-core v1.0.11-0.20240424194431-3612a5a6fb4c
    github.com/xtls/xray-core v1.260327.1-0.20260601021109-94ffd50060f1
)

replace (
    # qpack: v0.4.1 không tồn tại, ép về v0.4.0 hoặc v0.4.2
    github.com/quic-go/qpack => github.com/quic-go/qpack v0.4.2

    # nếu appernet/quic-go gây xung đột, ép về quic-go chính thức
    github.com/apernet/quic-go => github.com/quic-go/quic-go v0.39.0

    # protobuf thường bị lệch version
    google.golang.org/protobuf => google.golang.org/protobuf v1.36.0

    # grpc cũng nên ép về version ổn định
    google.golang.org/grpc => google.golang.org/grpc v1.81.0
)
