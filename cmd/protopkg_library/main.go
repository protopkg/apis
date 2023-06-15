package main

import (
	"flag"
	"fmt"
	"os"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	pppb "github.com/stackb/apis/build/stack/protobuf/package/v1alpha1"
)

var (
	protoset  = flag.String("protoset", "", "path to the compiled FileDescriptoSet")
	labelRepo = flag.String("repo", "", "bazel label repo")
	labelPkg  = flag.String("pkg", "", "bazel label package")
	labelName = flag.String("name", "", "bazel label name")
	output    = flag.String("output", "", "path of file to write the generated protoasset")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Println(err)
	}
}

func run() error {
	if *output == "" {
		return fmt.Errorf("-output file must be defined")
	}

	var ds descriptorpb.FileDescriptorSet
	data, err := os.ReadFile(*protoset)
	if err != nil {
		return fmt.Errorf("reading protoset file: %w", err)
	}

	if err := proto.Unmarshal(data, &ds); err != nil {
		return fmt.Errorf("unmarshaling protoset file: %w", err)
	}

	assets := make([]*pppb.ProtoAsset, len(ds.File))
	for _, file := range ds.File {
		asset, err := makeProtoAsset(file)
		if err != nil {
			return fmt.Errorf("making asset for %s: %v", file.Name, err)
		}
		assets = append(assets, asset)
	}

	pkg := &pppb.ProtoPackage{
		Assets: assets,
	}

	data, err = proto.Marshal(pkg)
	if err != nil {
		return fmt.Errorf("marshaling generated data: %v", err)
	}

	if err := os.WriteFile(*output, data, os.ModePerm); err != nil {
		return fmt.Errorf("writing generated file: %w", err)
	}

	return nil
}

func makeProtoAsset(file *descriptorpb.FileDescriptorProto) (*pppb.ProtoAsset, error) {
	return &pppb.ProtoAsset{
		File: file,
	}, nil
}
