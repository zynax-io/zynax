// SPDX-License-Identifier: Apache-2.0
module github.com/zynax-io/zynax/agents/adapters/git

go 1.26.3

replace github.com/zynax-io/zynax/protos/generated/go => ../../../protos/generated/go

require gopkg.in/yaml.v3 v3.0.1

require (
	github.com/google/go-github/v67 v67.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/zynax-io/zynax/protos/generated/go v0.0.0-20260526183321-7ed35c24f544 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260406210006-6f92a3bedf2d // indirect
	google.golang.org/grpc v1.81.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
