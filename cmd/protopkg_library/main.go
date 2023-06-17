package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	pppb "github.com/stackb/apis/build/stack/protobuf/package/v1alpha1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type flagName string

const (
	protoCompilerInfoFileFlagName    flagName = "proto_compiler_info_file"
	protoCompilerVersionFileFlagName flagName = "proto_compiler_version_file"
	protoDescriptorSetFileFlagName   flagName = "proto_descriptor_set_file"
	protoRepositoryInfoFileFlagName  flagName = "proto_repository_info_file"
	protoOutputFileFlagName          flagName = "proto_out"
	jsonOutputFileFlagName           flagName = "json_out"
)

var (
	protoCompilerInfoFile    = flag.String(string(protoCompilerInfoFileFlagName), "", "path to the proto_compiler info file")
	protoCompilerVersionFile = flag.String(string(protoCompilerVersionFileFlagName), "", "path to the proto_compiler version file")
	protoDescriptorSetFile   = flag.String(string(protoDescriptorSetFileFlagName), "", "path to the compiled FileDescriptoSet")
	protoRepositoryInfoFile  = flag.String(string(protoRepositoryInfoFileFlagName), "", "path to the proto_repository_info file")
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

	repository, err := readProtoRepositoryInfoFile(protoRepositoryInfoFileFlagName, *protoRepositoryInfoFile)
	if err != nil {
		return err
	}

	compiler, err := readProtoCompilerInfoFile(protoCompilerInfoFileFlagName, *protoCompilerInfoFile)
	if err != nil {
		return err
	}

	version, err := readProtoCompilerVersionFile(protoRepositoryInfoFileFlagName, *protoCompilerVersionFile)
	if err != nil {
		return err
	}

	pkg, err := makeProtoPackage(protoDescriptorSetData, protoDescriptorSet, repository, compiler, version)
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

func readProtoDescriptorSetFile(flag flagName, filename string) (*descriptorpb.FileDescriptorSet, []byte, error) {
	if filename == "" {
		return nil, nil, fmt.Errorf("flag required but not provided: %s", flag)
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
		return "", fmt.Errorf("flag required but not provided: %s", flag)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", flag, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func readProtoCompilerInfoFile(flag flagName, filename string) (*ProtoCompilerInfo, error) {
	if filename == "" {
		return nil, fmt.Errorf("flag required but not provided: %s", flag)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", flag, err)
	}
	var info ProtoCompilerInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("unmarshaling %s: %w", flag, err)
	}
	return &info, nil
}

func readProtoRepositoryInfoFile(flag flagName, filename string) (*ProtoRepositoryInfo, error) {
	if filename == "" {
		return nil, fmt.Errorf("flag required but not provided: %s", flag)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", flag, err)
	}
	var info ProtoRepositoryInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("unmarshaling %s: %w", flag, err)
	}
	return &info, nil
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
	repositoryInfo *ProtoRepositoryInfo,
	compilerInfo *ProtoCompilerInfo,
	compilerVersion string,
) (*pppb.ProtoPackage, error) {
	if repositoryInfo.SourceHost == "" {
		return nil, fmt.Errorf("repository source_host is required")
	}
	if repositoryInfo.SourceOwner == "" {
		return nil, fmt.Errorf("repository source_owner is required")
	}
	if repositoryInfo.SourceRepo == "" {
		return nil, fmt.Errorf("repository source_name is required")
	}
	if repositoryInfo.SourceCommit == "" {
		return nil, fmt.Errorf("repository source_commit is required")
	}

	assets := make([]*pppb.ProtoAsset, len(ds.File))
	for i, file := range ds.File {
		var asset pppb.ProtoAsset
		asset.File = file
		asset.Sha256 = sha256Bytes(data)
		asset.Size = uint64(len(data))
		assets[i] = &asset
	}

	pkg := &pppb.ProtoPackage{
		Location: &pppb.ProtoSourceLocation{
			Repository: &pppb.ProtoRepository{
				Host:       repositoryInfo.SourceHost,
				Name:       repositoryInfo.SourceRepo,
				Owner:      repositoryInfo.SourceOwner,
				Repository: fmt.Sprintf("%s/%s/%s", repositoryInfo.SourceHost, repositoryInfo.SourceOwner, repositoryInfo.SourceRepo),
			},
			Commit: repositoryInfo.SourceCommit,
			Prefix: repositoryInfo.SourcePrefix,
		},
		Compiler: &pppb.ProtoCompiler{
			Name:    compilerInfo.Name,
			Version: compilerVersion,
		},
		Assets: assets,
	}

	prefix := pkg.Location.Prefix
	if prefix == "" {
		prefix = "~"
	}

	pkg.Name = fmt.Sprintf("%s/%s/%s/%s@%s",
		pkg.Location.Repository.Host,
		pkg.Location.Repository.Owner,
		pkg.Location.Repository.Name,
		prefix,
		pkg.Location.Commit,
	)

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
