module github.com/handcraftdev/blackhole/core/pkg/plugins/node

go 1.21

require (
    github.com/handcraftdev/blackhole/core v0.0.0-00010101000000-000000000000
    github.com/handcraftdev/blackhole/core/pkg/sdk/plugin v0.0.0-00010101000000-000000000000
    google.golang.org/grpc v1.59.0
    google.golang.org/protobuf v1.31.0
)

replace (
    github.com/handcraftdev/blackhole/core => ../../../
    github.com/handcraftdev/blackhole/core/pkg/sdk/plugin => ../../sdk/plugin
)