module github.com/gosrvlibexample/gosrvlibexample

go 1.16

replace github.com/nexmoinc/gosrvlib => ../..

require (
	github.com/nexmoinc/gosrvlib v1.8.1
	github.com/prometheus/client_golang v1.11.0
	github.com/rakyll/gotest v0.0.6 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.18.1
)
