module github.com/zrma/proglog

go 1.26.2

require (
	github.com/casbin/casbin/v2 v2.105.0
	github.com/edsrzf/mmap-go v1.2.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/hashicorp/raft v1.7.3
	github.com/hashicorp/raft-boltdb v0.0.0-20250225060035-8f7048cdfa53
	github.com/hashicorp/serf v0.10.2
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.11.1
	github.com/travisjeffery/go-dynaport v1.0.0
	go.opencensus.io v0.24.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.8.1 // indirect
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/casbin/govaluate v1.7.0 // indirect
	github.com/cloudflare/cfssl v1.6.5 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-sql-driver/mysql v1.9.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/certificate-transparency-go v1.3.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-hclog v1.6.2 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-metrics v0.5.4 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-msgpack/v2 v2.1.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-sockaddr v1.0.7 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/memberlist v0.5.3 // indirect
	github.com/jmhodges/clock v1.2.0 // indirect
	github.com/jmoiron/sqlx v1.4.0 // indirect
	github.com/kisielk/sqlstruct v0.0.0-20210630145711-dae28ed37023 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.28 // indirect
	github.com/miekg/dns v1.1.66 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sean-/seed v0.0.0-20170313163322-e2103e2c3529 // indirect
	github.com/weppos/publicsuffix-go v0.40.3-0.20250408071509-6074bbe7fd39 // indirect
	github.com/zmap/zcrypto v0.0.0-20250418211859-7510c141e4b7 // indirect
	github.com/zmap/zlint/v3 v3.6.6 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/mod v0.34.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	golang.org/x/tools v0.43.0 // indirect
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.5.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
)

tool (
	github.com/cloudflare/cfssl/cmd/cfssl
	github.com/cloudflare/cfssl/cmd/cfssljson
	google.golang.org/grpc/cmd/protoc-gen-go-grpc
	google.golang.org/protobuf/cmd/protoc-gen-go
)

exclude (
	github.com/armon/go-metrics v0.4.2
	github.com/armon/go-metrics v0.5.0
	github.com/armon/go-metrics v0.5.1
	github.com/armon/go-metrics v0.5.2
	github.com/armon/go-metrics v0.5.3
	github.com/armon/go-metrics v0.5.4
)
