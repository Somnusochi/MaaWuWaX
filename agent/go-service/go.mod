module github.com/MaaWuWaX/MaaWuWaX/agent/go-service

go 1.25.0

require (
	github.com/MaaXYZ/maa-framework-go/v4 v4.0.0-beta.17
	github.com/bytedance/sonic v1.15.0
	github.com/rs/zerolog v1.34.0
)

require (
	github.com/bytedance/gopkg v0.1.3 // indirect
	github.com/bytedance/sonic/loader v0.5.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	golang.org/x/arch v0.25.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
)

replace github.com/ebitengine/purego => github.com/ebitengine/purego v0.9.1 // indirect; pinned for maa-framework-go compatibility
