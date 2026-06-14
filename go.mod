module github.com/caothemanh/psiphon_v2ray

go 1.26

require (
    github.com/Psiphon-Labs/psiphon-tunnel-core v1.0.11-0.20240424194431-3612a5a6fb4c
    github.com/xtls/xray-core v1.260327.1-0.20260601021109-94ffd50060f1
)

# Fix dependency lỗi
replace github.com/quic-go/qpack => github.com/quic-go/qpack v0.4.2
replace github.com/apernet/quic-go => github.com/quic-go/quic-go v0.39.0
replace google.golang.org/protobuf => google.golang.org/protobuf v1.36.0
replace google.golang.org/grpc => google.golang.org/grpc v1.81.0
