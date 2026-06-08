// SPDX-License-Identifier: Apache-2.0

module github.com/zynax-io/zynax/services/memory-service

go 1.26.4

require (
	github.com/alicebob/miniredis/v2 v2.38.0
	github.com/redis/go-redis/v9 v9.20.0
	github.com/zynax-io/zynax/libs/zynaxconfig v0.0.0-00010101000000-000000000000
	github.com/zynax-io/zynax/protos/generated/go v0.0.0
	google.golang.org/grpc v1.81.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260406210006-6f92a3bedf2d // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/zynax-io/zynax/libs/zynaxconfig => ../../libs/zynaxconfig

replace github.com/zynax-io/zynax/protos/generated/go => ../../protos/generated/go
