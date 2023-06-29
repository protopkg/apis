package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	gzflag "github.com/bazelbuild/bazel-gazelle/flag"
	pppb "github.com/stackb/apis/build/stack/protobuf/package/v1alpha2"
	"github.com/stackb/protoreflecthash"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type flagName string

const (
	depFlagName             flagName = "dep"
	protoOutputFileFlagName flagName = "proto_out"
	jsonOutputFileFlagName  flagName = "json_out"
)

var (
	protoOutputFile = flag.String(string(protoOutputFileFlagName), "", "path of file to write the generated proto file")
	jsonOutputFile  = flag.String(string(jsonOutputFileFlagName), "", "path of file to write the generated json file")
	depFiles        []string
)

func main() {
	flag.Var(&gzflag.MultiFlag{Values: &depFiles}, string(depFlagName), "path to proto_file dep output file (repeatable)")

	if err := run(); err != nil {
		fmt.Println(err)
	}
}

func run() error {
	flag.Parse()

	var fileDeps []*pppb.ProtoPackage
	for _, filename := range depFiles {
		fileDep, err := readProtoPackageFile(depFlagName, filename)
		if err != nil {
			return err
		}
		fileDeps = append(fileDeps, fileDep)
	}

	pkg, err := makeProtoPackage(fileDeps)
	if err != nil {
		return err
	}

	if *protoOutputFile != "" {
		if err := writeProtoOutputFile(pkg, *protoOutputFile); err != nil {
			return err
		}
	}
	if *jsonOutputFile != "" {
		if err := writeJsonOutputFile(pkg, *jsonOutputFile); err != nil {
			return err
		}
	}

	return nil
}

func readProtoPackageFile(flag flagName, filename string) (*pppb.ProtoPackage, error) {
	if filename == "" {
		return nil, errorFlagRequired(flag)
	}

	var pp pppb.ProtoPackage
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", flag, err)
	}
	if err := proto.Unmarshal(data, &pp); err != nil {
		return nil, fmt.Errorf("unmarshaling %s: %w", flag, err)
	}
	return &pp, nil
}

func writeProtoOutputFile(msg proto.Message, filename string) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling generated data: %v", err)
	}
	if err := os.WriteFile(filename, data, os.ModePerm); err != nil {
		return fmt.Errorf("writing proto file: %w", err)
	}
	return nil
}

func writeJsonOutputFile(msg proto.Message, filename string) error {
	marshaler := protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}
	jsonstr, err := marshaler.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling json: %w", err)
	}
	if err := os.WriteFile(filename, []byte(jsonstr), os.ModePerm); err != nil {
		return fmt.Errorf("writing json file: %w", err)
	}
	return nil
}

func makeProtoPackage(deps []*pppb.ProtoPackage) (*pppb.ProtoPackage, error) {

	protoFiles := make([]*pppb.ProtoFile, 0, len(deps))
	for _, pkg := range deps {
		protoFiles = append(protoFiles, pkg.Files...)
	}
	sort.Slice(protoFiles, func(i, j int) bool {
		a := protoFiles[i]
		b := protoFiles[j]
		return *a.File.Name < *b.File.Name
	})

	hash, err := makeProtoPackageHash(protoFiles)
	if err != nil {
		return nil, fmt.Errorf("calculating proto package hash: %w", err)
	}

	rep := deps[0]
	pkg := &pppb.ProtoPackage{
		Archive:  rep.Archive,
		Compiler: rep.Compiler,
		Files:    protoFiles,
		Hash:     hash,
	}
	pkg.Name = *rep.Files[0].File.Package

	return pkg, nil
}

func errorFlagRequired(name flagName) error {
	return fmt.Errorf("flag required but not provided: -%s", name)
}

func protoreflectHash(msg proto.Message) (string, error) {
	hasher := protoreflecthash.NewHasher()
	data, err := hasher.HashProto(msg.ProtoReflect())
	if err != nil {
		return "", fmt.Errorf("hashing proto: %w", err)
	}
	return fmt.Sprintf("protoreflecthash.v0:%x", data), nil
}

func makeProtoPackageHash(files []*pppb.ProtoFile) (string, error) {
	return protoreflectHash(&pppb.ProtoPackage{
		Files: files,
	})
}
