package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	pppb "github.com/stackb/apis/build/stack/protobuf/package/v1alpha1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	protoCompilerInfoFileFlagName    = "proto_compiler_info_file"
	protoCompilerVersionFileFlagName = "proto_compiler_version_file"
	protoDescriptorSetFileFlagName   = "proto_descriptor_set_file"
	protoRepositoryInfoFileFlagName  = "proto_repository_info_file"
	protoOutputFileFlagName          = "proto_out"
	jsonOutputFileFlagName           = "json_out"
)

var (
	protoCompilerInfoFile    = flag.String(protoCompilerInfoFileFlagName, "", "path to the proto_compiler info file")
	protoCompilerVersionFile = flag.String(protoCompilerVersionFileFlagName, "", "path to the proto_compiler version file")
	protoDescriptorSetFile   = flag.String(protoDescriptorSetFileFlagName, "", "path to the compiled FileDescriptoSet")
	protoRepositoryInfoFile  = flag.String(protoRepositoryInfoFileFlagName, "", "path to the proto_repository_info file")
	protoOutputFile          = flag.String(protoOutputFileFlagName, "", "path of file to write the generated proto file")
	jsonOutputFile           = flag.String(jsonOutputFileFlagName, "", "path of file to write the generated json file")
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

	if err := writeOutputFiles(pkg); err != nil {
		return err
	}

	return nil
}

func readProtoDescriptorSetFile(flagName, filename string) (*descriptorpb.FileDescriptorSet, []byte, error) {
	if filename == "" {
		return nil, nil, fmt.Errorf("flag required but not provided: %s", flagName)
	}
	var ds descriptorpb.FileDescriptorSet
	data, err := os.ReadFile(*protoDescriptorSetFile)
	if err != nil {
		return nil, nil, fmt.Errorf("reading proto_descriptor_set_file: %w", err)
	}
	if err := proto.Unmarshal(data, &ds); err != nil {
		return nil, nil, fmt.Errorf("unmarshaling proto_descriptor_set_file: %w", err)
	}
	return &ds, data, nil
}

func readProtoCompilerVersionFile(flagName, filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("flag required but not provided: %s", flagName)
	}

	data, err := os.ReadFile(*protoCompilerVersionFile)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", flagName, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func readProtoCompilerInfoFile(flagName, filename string) (*ProtoCompilerInfo, error) {
	if filename == "" {
		return nil, fmt.Errorf("flag required but not provided: %s", flagName)
	}
	data, err := os.ReadFile(*protoCompilerInfoFile)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", flagName, err)
	}
	log.Println("compiler-info:", string(data))
	var info ProtoCompilerInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("unmarshaling %s: %w", flagName, err)
	}
	return &info, nil
}

func readProtoRepositoryInfoFile(flagName, filename string) (*ProtoRepositoryInfo, error) {
	if filename == "" {
		return nil, fmt.Errorf("flag required but not provided: %s", flagName)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", flagName, err)
	}
	var info ProtoRepositoryInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("unmarshaling %s: %w", flagName, err)
	}
	return &info, nil
}

func writeOutputFiles(pkg *pppb.ProtoPackage) error {
	data, err := proto.Marshal(pkg)
	if err != nil {
		return fmt.Errorf("marshaling generated data: %v", err)
	}

	if *protoOutputFile != "" {
		if err := os.WriteFile(*protoOutputFile, data, os.ModePerm); err != nil {
			return fmt.Errorf("writing proto file: %w", err)
		}
	}

	if *jsonOutputFile != "" {
		marshaler := protojson.MarshalOptions{
			Multiline: true,
			Indent:    "  ",
		}
		jsonstr, err := marshaler.Marshal(pkg)
		if err != nil {
			return fmt.Errorf("marshaling json: %w", err)
		}
		if err := os.WriteFile(*jsonOutputFile, []byte(jsonstr), os.ModePerm); err != nil {
			return fmt.Errorf("writing json file: %w", err)
		}
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
		return nil, fmt.Errorf("SourceHost is required")
	}
	if repositoryInfo.SourceOwner == "" {
		return nil, fmt.Errorf("SourceOwner is required")
	}
	if repositoryInfo.SourceRepo == "" {
		return nil, fmt.Errorf("SourceName is required")
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
				Server:     repositoryInfo.SourceHost,
				Name:       repositoryInfo.SourceRepo,
				Owner:      repositoryInfo.SourceOwner,
				Repository: fmt.Sprintf("%s/%s/%s", repositoryInfo.SourceHost, repositoryInfo.SourceOwner, repositoryInfo.SourceRepo),
			},
			Commit: repositoryInfo.Commit,
			Prefix: repositoryInfo.SourcePrefix,
		},
		Compiler: &pppb.ProtoCompiler{
			Name:    compilerInfo.Name,
			Version: compilerVersion,
		},
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
