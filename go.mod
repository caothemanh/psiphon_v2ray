module github.com/caothemanh/psiphon_v2ray

go 1.21

replace (
    github.com/Psiphon-Labs/chacha20 => github.com/Psiphon-Labs/chacha20 v0.0.0-20200916121732-6a9e25bdf1f7
    github.com/Psiphon-Labs/goptlib => github.com/Psiphon-Labs/goptlib v0.0.0-20200406165125-c0e32a7a3464
)

require (
	github.com/Psiphon-Labs/bolt v0.0.0-20200624191537-23cedaef7ad7 // indirect
	github.com/Psiphon-Labs/chacha20 v0.0.0-20181203154727-3a73f2894dbf // indirect
	github.com/Psiphon-Labs/compress v0.0.0-20230918195954-dda6b7e7ef98 // indirect
	github.com/Psiphon-Labs/goptlib v0.0.0-20200406165125-c0e32a7a3464 // indirect
	github.com/Psiphon-Labs/psiphon-tls v0.0.0-20240305030049-8e5cc3b71a8e // indirect
	github.com/Psiphon-Labs/quic-go v0.0.0-20240305032007-8bef59fc3db5 // indirect
	github.com/Psiphon-Labs/rotate v0.0.0-20210601003148-9f835fc6cbf5 // indirect
	github.com/Psiphon-Labs/tun2socks v1.16.11-0.20220723025548-bf8cff848c85 // indirect
)
