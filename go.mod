module github.com/protopkg/apis

go 1.19

require (
	github.com/bazelbuild/bazel-gazelle v0.31.1
	github.com/stackb/rules_proto v0.0.0-20230612182459-3d7eec0c990f
)

require (
	github.com/bazelbuild/buildtools v0.0.0-20230510134650-37bd1811516d // indirect
	github.com/emicklei/proto v1.9.0 // indirect
	go.starlark.net v0.0.0-20220328144851-d1966c6b9fcd // indirect
	golang.org/x/mod v0.10.0 // indirect
	golang.org/x/sys v0.9.0 // indirect
	golang.org/x/tools v0.9.1 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace (
	github.com/stackb/apis/build/stack/protobuf/package/v1alpha1 => ../../stackb/apis/bazel-bin/build/stack/protobuf/package/v1alpha1/go.mod
)
