module github.com/caothemanh/psiphon_v2ray

go 1.21

require (
	github.com/Psiphon-Labs/psiphon-tunnel-core v2.0.28+incompatible
	github.com/xtls/xray-core v1.8.24
	golang.org/x/mobile v0.0.0-20240506190922-a749a6e2b1b2
)

tool (
	golang.org/x/mobile/cmd/gobind
	golang.org/x/mobile/cmd/gomobile
)
