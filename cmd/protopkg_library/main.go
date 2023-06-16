package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	pppb "github.com/stackb/apis/build/stack/protobuf/package/v1alpha1"
)

var (
	labelRepo = flag.String("repo", "", "bazel label repo")
	labelPkg  = flag.String("pkg", "", "bazel label package")
	labelName = flag.String("name", "", "bazel label name")

	protosetInFile = flag.String("protoset", "", "path to the compiled FileDescriptoSet")
	protoOutFile   = flag.String("proto_out", "", "path of file to write the generated proto file")
	jsonOutFile    = flag.String("json_out", "", "path of file to write the generated json file")
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
	}
}

func run() error {
	flag.Parse()

	if *protosetInFile == "" {
		log.Fatal("-protoset file is required")
	}

	var ds descriptorpb.FileDescriptorSet
	data, err := os.ReadFile(*protosetInFile)
	if err != nil {
		return fmt.Errorf("reading protoset file: %w", err)
	}
	if err := proto.Unmarshal(data, &ds); err != nil {
		return fmt.Errorf("unmarshaling protoset file: %w", err)
	}

	pkg, err := makeProtoPackage(data, &ds)
	if err != nil {
		return err
	}

	if err := writeOutputFiles(pkg); err != nil {
		return err
	}

	return nil
}

func writeOutputFiles(pkg *pppb.ProtoPackage) error {
	data, err := proto.Marshal(pkg)
	if err != nil {
		return fmt.Errorf("marshaling generated data: %v", err)
	}

	if *protoOutFile != "" {
		if err := os.WriteFile(*protoOutFile, data, os.ModePerm); err != nil {
			return fmt.Errorf("writing proto file: %w", err)
		}
	}

	if *jsonOutFile != "" {
		marshaler := protojson.MarshalOptions{
			Multiline: true,
			Indent:    "  ",
		}
		jsonstr, err := marshaler.Marshal(pkg)
		if err != nil {
			return fmt.Errorf("marshaling json: %w", err)
		}
		if err := os.WriteFile(*jsonOutFile, []byte(jsonstr), os.ModePerm); err != nil {
			return fmt.Errorf("writing json file: %w", err)
		}
	}
	return nil
}

func makeProtoPackage(data []byte, ds *descriptorpb.FileDescriptorSet) (*pppb.ProtoPackage, error) {
	assets := make([]*pppb.ProtoAsset, len(ds.File))
	for _, file := range ds.File {
		var asset pppb.ProtoAsset
		asset.File = file
		asset.Sha256 = sha256Bytes(data)
		asset.Size = uint64(len(data))
		assets = append(assets, &asset)
	}

	pkg := &pppb.ProtoPackage{
		Assets: assets,
	}

	return pkg, nil
}

func makeProtoAsset(file *descriptorpb.FileDescriptorProto) (*pppb.ProtoAsset, error) {
	return &pppb.ProtoAsset{
		File: file,
	}, nil
}

func sha256Bytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
