module github.com/zynax-io/zynax/protos/tests

go 1.24.0

require (
	github.com/cucumber/godog v0.14.1
	github.com/zynax-io/zynax/protos/generated/go v0.0.0
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.10
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cucumber/gherkin/go/v26 v26.2.0 // indirect
	github.com/cucumber/messages/go/v21 v21.0.1 // indirect
	github.com/gofrs/uuid v4.3.1+incompatible // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-memdb v1.3.4 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
)

replace github.com/zynax-io/zynax/protos/generated/go => ../generated/go
