module github.com/apernet/quic-go

go 1.21

require (
    github.com/stretchr/testify v1.11.1
    github.com/onsi/ginkgo/v2 v2.9.5
    github.com/onsi/gomega v1.27.6
    github.com/quic-go/qpack v0.4.0
    golang.org/x/net v0.55.0
    golang.org/x/sys v0.45.0
    golang.org/x/crypto v0.51.0
    golang.org/x/text v0.37.0
    golang.org/x/sync v0.20.0
)

replace github.com/quic-go/qpack => github.com/quic-go/qpack v0.4.0
