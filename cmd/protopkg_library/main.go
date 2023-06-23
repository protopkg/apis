package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	pppb "github.com/stackb/apis/build/stack/protobuf/package/v1alpha1"
	"github.com/stackb/protoreflecthash"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type flagName string

const (
	protoCompilerNameFlagName                  flagName = "proto_compiler_name"
	protoCompilerVersionFileFlagName           flagName = "proto_compiler_version_file"
	protoDescriptorSetFileFlagName             flagName = "proto_descriptor_set_file"
	protoRepositoryHostFlagName                flagName = "proto_repository_host"
	protoRepositoryOwnerFlagName               flagName = "proto_repository_owner"
	protoRepositoryRepoFlagName                flagName = "proto_repository_repo"
	protoRepositoryCommitFlagName              flagName = "proto_repository_commit"
	protoRepositoryPrefixFlagName              flagName = "proto_repository_prefix"
	protoPackageDirectDependenciesFileFlagName flagName = "proto_package_direct_dependency_files"
	protoOutputFileFlagName                    flagName = "proto_out"
	jsonOutputFileFlagName                     flagName = "json_out"
)

var (
	protoCompilerName                    = flag.String(string(protoCompilerNameFlagName), "", "proto compiler name")
	protoCompilerVersionFile             = flag.String(string(protoCompilerVersionFileFlagName), "", "path to the proto_compiler version file")
	protoDescriptorSetFile               = flag.String(string(protoDescriptorSetFileFlagName), "", "path to the compiled FileDescriptoSet")
	protoPackageSetDirectDependencyFiles = flag.String(string(protoPackageDirectDependenciesFileFlagName), "", "comma-separated path list to a proto packages that represents the direct package dependencies of this one")
	protoRepositoryHost                  = flag.String(string(protoRepositoryHostFlagName), "", "value of the proto_repository.host")
	protoRepositoryOwner                 = flag.String(string(protoRepositoryOwnerFlagName), "", "value of the proto_repository.owner")
	protoRepositoryRepo                  = flag.String(string(protoRepositoryRepoFlagName), "", "value of the proto_repository.repo")
	protoRepositoryCommit                = flag.String(string(protoRepositoryCommitFlagName), "", "value of the proto_repository.commit")
	protoRepositoryPrefix                = flag.String(string(protoRepositoryPrefixFlagName), "", "value of the proto_repository.prefix")
	protoOutputFile                      = flag.String(string(protoOutputFileFlagName), "", "path of file to write the generated proto file")
	jsonOutputFile                       = flag.String(string(jsonOutputFileFlagName), "", "path of file to write the generated json file")
)

var (
	assetDeps   = make(map[string]*pppb.ProtoAsset)
	packageDeps = make(map[string]*pppb.ProtoPackage)
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
	}
}

func run() error {
	flag.Parse()

	deps, err := readProtoPackageSetDirectDependencies(protoPackageDirectDependenciesFileFlagName, *protoPackageSetDirectDependencyFiles)
	if err != nil {
		return err
	}
	collectPackageDeps(deps)

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

func readProtoPackageSetDirectDependencies(flag flagName, commaSeparatedfilenames string) (*pppb.ProtoPackageSet, error) {
	var ps pppb.ProtoPackageSet
	if commaSeparatedfilenames != "" {
		filenames := strings.Split(commaSeparatedfilenames, ",")
		for _, filename := range filenames {
			pkg, err := readProtoPackageFile(flag, filename)
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", filename, err)
			}
			ps.Packages = append(ps.Packages, pkg)
		}
	}
	return &ps, nil
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

func readProtoDescriptorSetFile(flag flagName, filename string) (*descriptorpb.FileDescriptorSet, []byte, error) {
	if filename == "" {
		return nil, nil, errorFlagRequired(flag)
	}

	var ds descriptorpb.FileDescriptorSet
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", flag, err)
	}
	if err := proto.Unmarshal(data, &ds); err != nil {
		return nil, nil, fmt.Errorf("unmarshaling %s: %w", flag, err)
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

	hash, err := makeProtoPackageHash(assets)
	if err != nil {
		return nil, fmt.Errorf("calculating proto package hash: %w", err)
	}

	pkg := &pppb.ProtoPackage{
		Location:     location,
		Compiler:     compiler,
		Assets:       assets,
		Hash:         hash,
		Dependencies: makeProtoPackageDependencies(),
	}
	pkg.Name = makeProtoPackageName(pkg)

	return pkg, nil
}

func makeProtoPackageName(pkg *pppb.ProtoPackage) string {
	name := path.Join(pkg.Location.Repository.Repository, pkg.Location.Prefix)
	if len(pkg.Assets) == 1 {
		name = name + ":" + *pkg.Assets[0].File.Name
	}
	return "//" + name + "@" + pkg.Location.Commit
}

func makeProtoAsset(file *descriptorpb.FileDescriptorProto) (*pppb.ProtoAsset, error) {
	sortFile(file)

	data, err := proto.Marshal(file)
	if err != nil {
		return nil, fmt.Errorf("marshaling asset FileDescriptorProto: %w", err)
	}
	hash, err := protoreflectHash(file)
	if err != nil {
		return nil, fmt.Errorf("calculating fileset hash: %w", err)
	}
	deps, err := makeProtoAssetDependencies(file.Dependency)
	if err != nil {
		return nil, fmt.Errorf("assembling file deps: %w", err)
	}

	return &pppb.ProtoAsset{
		File:         file,
		Sha256:       sha256Bytes(data),
		Size:         uint64(len(data)),
		Hash:         hash,
		Dependencies: deps,
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

func makeProtoAssetDependencies(deps []string) ([]string, error) {
	results := make([]string, len(deps))
	for i, dep := range deps {
		asset, ok := assetDeps[dep]
		if !ok {
			return nil, fmt.Errorf("asset dependency not found: %s", dep)
		}
		results[i] = assetHashKey(asset)
	}
	sort.Strings(results)
	return results, nil
}

func sortFile(f *descriptorpb.FileDescriptorProto) {
	sort.Strings(f.Dependency)
	// TODO: fix the public dependencies here: not used, but still, they shouold
	// continue to correlate with the f.Dependency above.
	sort.Slice(f.EnumType, func(i, j int) bool {
		a := f.EnumType[i]
		b := f.EnumType[j]
		return *a.Name < *b.Name
	})
	sort.Slice(f.MessageType, func(i, j int) bool {
		a := f.MessageType[i]
		b := f.MessageType[j]
		return *a.Name < *b.Name
	})
	sort.Slice(f.Service, func(i, j int) bool {
		a := f.Service[i]
		b := f.Service[j]
		return *a.Name < *b.Name
	})
	sort.Slice(f.Extension, func(i, j int) bool {
		a := f.Extension[i]
		b := f.Extension[j]
		return *a.Name < *b.Name
	})

	for _, e := range f.EnumType {
		sortEnumType(e)
	}
	for _, m := range f.MessageType {
		sortMessageType(m)
	}
	for _, s := range f.Service {
		sortService(s)
	}
	for _, e := range f.Extension {
		sortExtension(e)
	}

	if f.SourceCodeInfo != nil {
		sortSourceCodeInfo(f.SourceCodeInfo)
	}
	if f.Options != nil {
		sortFileOptions(f.Options)
	}
}

func sortEnumType(e *descriptorpb.EnumDescriptorProto) {
	sort.Slice(e.Value, func(i, j int) bool {
		a := e.Value[i]
		b := e.Value[j]
		return *a.Number < *b.Number
	})
	if e.Options != nil {
		sortEnumOptions(e.Options)
	}
}

func sortMessageType(m *descriptorpb.DescriptorProto) {
	sort.Strings(m.ReservedName)

	sort.Slice(m.EnumType, func(i, j int) bool {
		a := m.EnumType[i]
		b := m.EnumType[j]
		return *a.Name < *b.Name
	})
	sort.Slice(m.Field, func(i, j int) bool {
		a := m.Field[i]
		b := m.Field[j]
		return *a.Number < *b.Number
	})
	sort.Slice(m.NestedType, func(i, j int) bool {
		a := m.NestedType[i]
		b := m.NestedType[j]
		return *a.Name < *b.Name
	})
	sort.Slice(m.Extension, func(i, j int) bool {
		a := m.Extension[i]
		b := m.Extension[j]
		return *a.Name < *b.Name
	})
	sort.Slice(m.ExtensionRange, func(i, j int) bool {
		a := m.ExtensionRange[i]
		b := m.ExtensionRange[j]
		return *a.Start < *b.Start
	})
	sort.Slice(m.ReservedRange, func(i, j int) bool {
		a := m.ReservedRange[i]
		b := m.ReservedRange[j]
		return *a.Start < *b.Start
	})

	for _, e := range m.EnumType {
		sortEnumType(e)
	}
	for _, f := range m.Field {
		sortFieldType(f)
	}
	for _, m := range m.NestedType {
		sortMessageType(m)
	}
	for _, e := range m.Extension {
		sortExtension(e)
	}
	for _, e := range m.ExtensionRange {
		sortExtensionRange(e)
	}
	for _, e := range m.ReservedRange {
		sortReservedRange(e)
	}
}

func sortFieldType(f *descriptorpb.FieldDescriptorProto) {
	// TODO: f.Label?
	// TODO: f.Type?
	if f.Options != nil {
		sortFieldOptions(f.Options)
	}
}

func sortService(s *descriptorpb.ServiceDescriptorProto) {
	sort.Slice(s.Method, func(i, j int) bool {
		a := s.Method[i]
		b := s.Method[j]
		return *a.Name < *b.Name
	})
	for _, m := range s.Method {
		sortMethod(m)
	}
	if s.Options != nil {
		sortServiceOptions(s.Options)
	}
	// DONE
}

func sortMethod(s *descriptorpb.MethodDescriptorProto) {
	if s.Options != nil {
		sortMethodOptions(s.Options)
	}
	// DONE
}

func sortExtension(e *descriptorpb.FieldDescriptorProto) {
	sortFieldType(e)
	// DONE
}

func sortExtensionRange(e *descriptorpb.DescriptorProto_ExtensionRange) {
	if e.Options != nil {
		sortExtensionRangeOptions(e.Options)
	}
	// DONE
}

func sortReservedRange(e *descriptorpb.DescriptorProto_ReservedRange) {
	// DONE
}

func sortExtensionRangeOptions(e *descriptorpb.ExtensionRangeOptions) {
	sort.Slice(e.UninterpretedOption, func(i, j int) bool {
		a := e.UninterpretedOption[i]
		b := e.UninterpretedOption[j]
		return *a.IdentifierValue < *b.IdentifierValue
	})
	for _, o := range e.UninterpretedOption {
		sortUninterpretedOption(o)
	}
	// DONE
}

func sortFileOptions(o *descriptorpb.FileOptions) {
	sort.Slice(o.UninterpretedOption, func(i, j int) bool {
		a := o.UninterpretedOption[i]
		b := o.UninterpretedOption[j]
		return *a.IdentifierValue < *b.IdentifierValue
	})
	for _, o := range o.UninterpretedOption {
		sortUninterpretedOption(o)
	}
	// DONE
}

func sortEnumOptions(o *descriptorpb.EnumOptions) {
	sort.Slice(o.UninterpretedOption, func(i, j int) bool {
		a := o.UninterpretedOption[i]
		b := o.UninterpretedOption[j]
		return *a.IdentifierValue < *b.IdentifierValue
	})
	for _, o := range o.UninterpretedOption {
		sortUninterpretedOption(o)
	}
	// DONE
}

func sortFieldOptions(o *descriptorpb.FieldOptions) {
	// TODO: sort e.Retention?
	sort.Slice(o.UninterpretedOption, func(i, j int) bool {
		a := o.UninterpretedOption[i]
		b := o.UninterpretedOption[j]
		return *a.IdentifierValue < *b.IdentifierValue
	})
	for _, o := range o.UninterpretedOption {
		sortUninterpretedOption(o)
	}
	// DONE
}

func sortServiceOptions(o *descriptorpb.ServiceOptions) {
	sort.Slice(o.UninterpretedOption, func(i, j int) bool {
		a := o.UninterpretedOption[i]
		b := o.UninterpretedOption[j]
		return *a.IdentifierValue < *b.IdentifierValue
	})
	for _, o := range o.UninterpretedOption {
		sortUninterpretedOption(o)
	}
	// DONE
}

func sortMethodOptions(o *descriptorpb.MethodOptions) {
	sort.Slice(o.UninterpretedOption, func(i, j int) bool {
		a := o.UninterpretedOption[i]
		b := o.UninterpretedOption[j]
		return *a.IdentifierValue < *b.IdentifierValue
	})
	for _, o := range o.UninterpretedOption {
		sortUninterpretedOption(o)
	}
	// DONE
}

func sortUninterpretedOption(o *descriptorpb.UninterpretedOption) {
	// TODO: sort name parts?  Probably not.
}

func sortSourceCodeInfo(s *descriptorpb.SourceCodeInfo) {
	// TODO: sort this, or assume sorted?
}

func makeProtoPackageDependencies() []string {
	names := make([]string, 0, len(packageDeps))
	for _, pkg := range packageDeps {
		names = append(names, pkg.Name)
	}
	sort.Strings(names)
	return names
}

func collectPackageDeps(pps *pppb.ProtoPackageSet) {
	for _, pkg := range pps.Packages {
		packageDeps[packageHashKey(pkg)] = pkg
		for _, asset := range pkg.Assets {
			assetDeps[*asset.File.Name] = asset
		}
	}
}

func packageHashKey(pkg *pppb.ProtoPackage) string {
	return fmt.Sprintf("%s/%s@%s",
		pkg.Location.Repository.Repository,
		makePackagePrefix(pkg.Location.Prefix),
		pkg.Hash,
	)
}

func assetHashKey(asset *pppb.ProtoAsset) string {
	return fmt.Sprintf("%s@%s", *asset.File.Name, asset.Hash)
}

func makeProtoPackageHash(assets []*pppb.ProtoAsset) (string, error) {
	return protoreflectHash(&pppb.ProtoPackage{
		Assets: assets,
	})
}

func makePackagePrefix(prefix string) string {
	if prefix == "" {
		prefix = "~"
	}
	return prefix
}
