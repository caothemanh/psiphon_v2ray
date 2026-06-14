module github.com/caothemanh/psiphon_v2ray

go 1.21

require (
	github.com/Psiphon-Labs/psiphon-tunnel-core v1.0.11-0.20240424194431-3612a5a6fb4c
	github.com/xtls/xray-core v1.260327.1-0.20260601021109-94ffd50060f1
	golang.org/x/mobile v0.0.0-20260602190626-68735029466e
)

replace (
	github.com/xtls/xray-core => github.com/xtls/xray-core v1.260327.1-0.20260601021109-94ffd50060f1
	github.com/Psiphon-Labs/quic-go => github.com/Psiphon-Labs/quic-go v0.0.0-20240424181006-45545f5e1536
	github.com/quic-go/qpack => github.com/quic-go/qpack v0.4.1
)
