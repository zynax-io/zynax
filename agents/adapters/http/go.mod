// SPDX-License-Identifier: Apache-2.0
module github.com/zynax-io/zynax/agents/adapters/http

go 1.26.3

require (
	github.com/zynax-io/zynax/protos/generated/go v0.0.0
	google.golang.org/grpc v1.80.0
)

require (
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260406210006-6f92a3bedf2d // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/zynax-io/zynax/protos/generated/go => ../../../protos/generated/go
