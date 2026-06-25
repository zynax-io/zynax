// SPDX-License-Identifier: Apache-2.0
module github.com/zynax-io/zynax/agents/adapters/adk

go 1.26.4

replace github.com/zynax-io/zynax/protos/generated/go => ../../../protos/generated/go

require (
	github.com/zynax-io/zynax/protos/generated/go v0.0.0-20260526183321-7ed35c24f544
	google.golang.org/grpc v1.81.1
	google.golang.org/protobuf v1.36.11
	gopkg.in/yaml.v3 v3.0.1
)

require (
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260406210006-6f92a3bedf2d // indirect
)
