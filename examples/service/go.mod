module github.com/gosrvlibexample/gosrvlibexample

go 1.15

replace github.com/nexmoinc/gosrvlib => ../..

require (
	github.com/nexmoinc/gosrvlib v0.0.0-00010101000000-000000000000
	github.com/prometheus/client_golang v1.9.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.16.0
)
