module github.com/nexmoinc/gosrvlib-sample-service

go 1.15

replace github.com/nexmoinc/gosrvlib => ../..

require (
	github.com/golang/mock v1.4.4
	github.com/nexmoinc/gosrvlib v0.0.0-00010101000000-000000000000
	github.com/prometheus/client_golang v1.7.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.16.0
)
