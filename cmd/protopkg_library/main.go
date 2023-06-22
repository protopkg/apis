package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"

	pppb "github.com/stackb/apis/build/stack/protobuf/package/v1alpha1"
	"github.com/stackb/protoreflecthash"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type flagName string

const (
	protoCompilerNameFlagName        flagName = "proto_compiler_name"
	protoCompilerVersionFileFlagName flagName = "proto_compiler_version_file"
	protoDescriptorSetFileFlagName   flagName = "proto_descriptor_set_file"
	protoRepositoryHostFlagName      flagName = "proto_repository_host"
	protoRepositoryOwnerFlagName     flagName = "proto_repository_owner"
	protoRepositoryRepoFlagName      flagName = "proto_repository_repo"
	protoRepositoryCommitFlagName    flagName = "proto_repository_commit"
	protoRepositoryPrefixFlagName    flagName = "proto_repository_prefix"
	protoOutputFileFlagName          flagName = "proto_out"
	jsonOutputFileFlagName           flagName = "json_out"
)

var (
	protoCompilerName        = flag.String(string(protoCompilerNameFlagName), "", "proto compiler name")
	protoCompilerVersionFile = flag.String(string(protoCompilerVersionFileFlagName), "", "path to the proto_compiler version file")
	protoDescriptorSetFile   = flag.String(string(protoDescriptorSetFileFlagName), "", "path to the compiled FileDescriptoSet")
	protoRepositoryHost      = flag.String(string(protoRepositoryHostFlagName), "", "value of the proto_repository.host")
	protoRepositoryOwner     = flag.String(string(protoRepositoryOwnerFlagName), "", "value of the proto_repository.owner")
	protoRepositoryRepo      = flag.String(string(protoRepositoryRepoFlagName), "", "value of the proto_repository.repo")
	protoRepositoryCommit    = flag.String(string(protoRepositoryCommitFlagName), "", "value of the proto_repository.commit")
	protoRepositoryPrefix    = flag.String(string(protoRepositoryPrefixFlagName), "", "value of the proto_repository.prefix")
	protoOutputFile          = flag.String(string(protoOutputFileFlagName), "", "path of file to write the generated proto file")
	jsonOutputFile           = flag.String(string(jsonOutputFileFlagName), "", "path of file to write the generated json file")
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
	}
}

func run() error {
	flag.Parse()

	protoDescriptorSet, protoDescriptorSetData, err := readProtoDescriptorSetFile(protoDescriptorSetFileFlagName, *protoDescriptorSetFile)
	if err != nil {
		return err
	}

	version, err := readProtoCompilerVersionFile(protoCompilerVersionFileFlagName, *protoCompilerVersionFile)
	if err != nil {
		return err
	}

	compiler, err := makeProtoCompiler(version)
	if err != nil {
		return err
	}

	location, err := makeProtoSourceLocation()
	if err != nil {
		return err
	}

	pkg, err := makeProtoPackage(protoDescriptorSetData, protoDescriptorSet, location, compiler)
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

func makeProtoSourceLocation() (*pppb.ProtoSourceLocation, error) {
	if *protoRepositoryHost == "" {
		return nil, errorFlagRequired(protoRepositoryHostFlagName)
	}
	if *protoRepositoryOwner == "" {
		return nil, errorFlagRequired(protoRepositoryOwnerFlagName)
	}
	if *protoRepositoryRepo == "" {
		return nil, errorFlagRequired(protoRepositoryRepoFlagName)
	}
	if *protoRepositoryCommit == "" {
		return nil, errorFlagRequired(protoRepositoryCommitFlagName)
	}
	return &pppb.ProtoSourceLocation{
		Repository: &pppb.ProtoRepository{
			Host:       *protoRepositoryHost,
			Owner:      *protoRepositoryOwner,
			Name:       *protoRepositoryRepo,
			Repository: fmt.Sprintf("%s/%s/%s", *protoRepositoryHost, *protoRepositoryOwner, *protoRepositoryRepo),
		},
		Commit: *protoRepositoryCommit,
		Prefix: *protoRepositoryPrefix,
	}, nil
}

func makeProtoCompiler(version string) (*pppb.ProtoCompiler, error) {
	if *protoCompilerName == "" {
		return nil, errorFlagRequired(protoCompilerNameFlagName)
	}
	return &pppb.ProtoCompiler{
		Name:    *protoCompilerName,
		Version: version,
	}, nil
}

func readProtoDescriptorSetFile(flag flagName, filename string) (*descriptorpb.FileDescriptorSet, []byte, error) {
	if filename == "" {
		return nil, nil, errorFlagRequired(flag)
	}
	var ds descriptorpb.FileDescriptorSet
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("reading proto_descriptor_set_file: %w", err)
	}
	if err := proto.Unmarshal(data, &ds); err != nil {
		return nil, nil, fmt.Errorf("unmarshaling proto_descriptor_set_file: %w", err)
	}
	return &ds, data, nil
}

func readProtoCompilerVersionFile(flag flagName, filename string) (string, error) {
	if filename == "" {
		return "", errorFlagRequired(flag)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", flag, err)
	}
	return strings.TrimSpace(string(data)), nil
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

func makeProtoPackage(data []byte,
	ds *descriptorpb.FileDescriptorSet,
	location *pppb.ProtoSourceLocation,
	compiler *pppb.ProtoCompiler,
) (*pppb.ProtoPackage, error) {

	assets := make([]*pppb.ProtoAsset, len(ds.File))
	for i, file := range ds.File {
		asset, err := makeProtoAsset(file)
		if err != nil {
			return nil, fmt.Errorf("making ProtoAsset %d %s: %w", i, *file.Name, err)
		}
		assets[i] = asset
	}

	pkg := &pppb.ProtoPackage{
		Location: location,
		Compiler: compiler,
		Assets:   assets,
	}

	prefix := pkg.Location.Prefix
	if prefix == "" {
		prefix = "~"
	}

	pkg.Name = fmt.Sprintf("%s/%s/%s",
		pkg.Location.Repository.Repository,
		prefix,
		pkg.Location.Commit,
	)

	return pkg, nil
}

func makeProtoAsset(file *descriptorpb.FileDescriptorProto) (*pppb.ProtoAsset, error) {
	data, err := proto.Marshal(file)
	if err != nil {
		return nil, fmt.Errorf("marshaling asset FileDescriptorProto: %w", err)
	}
	hash, err := protoreflectHash(file)
	if err != nil {
		return nil, err
	}
	return &pppb.ProtoAsset{
		File:   file,
		Sha256: sha256Bytes(data),
		Size:   uint64(len(data)),
		Hash:   hash,
	}, nil
}

func sha256Bytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
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
