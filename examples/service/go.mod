module github.com/gosrvlibexample/gosrvlibexample

go 1.16

replace github.com/nexmoinc/gosrvlib => ../..

require (
	github.com/golang/mock v1.4.4 // indirect
	github.com/nexmoinc/gosrvlib v0.0.0-00010101000000-000000000000
	github.com/prometheus/client_golang v1.9.0
	github.com/rakyll/gotest v0.0.5 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.16.0
)
