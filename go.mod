module github.com/tecnickcom/gogen

go 1.26

tool go.uber.org/mock/mockgen

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/XSAM/otelsql v0.42.0
	github.com/aperturerobotics/go-brotli-decoder v1.2.2
	github.com/aws/aws-sdk-go-v2 v1.42.0
	github.com/aws/aws-sdk-go-v2/config v1.32.25
	github.com/aws/aws-sdk-go-v2/service/s3 v1.104.0
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.42.3
	github.com/aws/aws-sdk-go-v2/service/sqs v1.44.0
	github.com/aws/smithy-go v1.27.2
	github.com/confluentinc/confluent-kafka-go/v2 v2.14.2
	github.com/dlmiddlecote/sqlstats v1.0.2
	github.com/go-playground/validator/v10 v10.30.3
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/go-cmp v0.7.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/julienschmidt/httprouter v1.3.0
	github.com/prometheus/client_golang v1.23.2
	github.com/redis/go-redis/v9 v9.20.1
	github.com/rs/zerolog v1.35.1
	github.com/samber/slog-zerolog/v2 v2.9.2
	github.com/segmentio/kafka-go v0.4.51
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	github.com/spf13/viper/remote v1.21.0
	github.com/stretchr/testify v1.11.1
	github.com/tecnickcom/farmhash64 v1.9.72
	github.com/tecnickcom/statsd v1.0.71
	github.com/undefinedlabs/go-mpatch v1.0.7
	github.com/valkey-io/valkey-go v1.0.76
	github.com/valkey-io/valkey-go/mock v1.0.76
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.44.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.44.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.44.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.44.0
	go.opentelemetry.io/otel/metric v1.44.0
	go.opentelemetry.io/otel/sdk v1.44.0
	go.opentelemetry.io/otel/sdk/metric v1.44.0
	go.opentelemetry.io/otel/trace v1.44.0
	go.uber.org/mock v0.6.0
	go.uber.org/multierr v1.11.0
	golang.org/x/crypto v0.53.0
	golang.org/x/net v0.56.0
	golang.org/x/text v0.38.0
)

require (
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/auth v0.20.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/firestore v1.22.0 // indirect
	cloud.google.com/go/longrunning v1.0.0 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.13 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.24 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.2.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.31.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.36.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.43.3 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.7.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.16 // indirect
	github.com/googleapis/gax-go/v2 v2.22.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/hashicorp/consul/api v1.34.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-metrics v0.5.4 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/serf v0.10.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/nats.go v1.52.0 // indirect
	github.com/nats-io/nkeys v0.4.16 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.4.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.27 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.69.0 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/sagikazarmark/crypt v0.31.0 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/slog-common v0.22.0 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.etcd.io/etcd/api/v3 v3.6.12 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.6.12 // indirect
	go.etcd.io/etcd/client/v2 v2.305.31 // indirect
	go.etcd.io/etcd/client/v3 v3.6.12 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.69.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.44.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20260611194520-c48552f49976 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.46.0 // indirect
	google.golang.org/api v0.285.0 // indirect
	google.golang.org/genproto v0.0.0-20260618152121-87f3d3e198d3 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260618152121-87f3d3e198d3 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260618152121-87f3d3e198d3 // indirect
	google.golang.org/grpc v1.81.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
