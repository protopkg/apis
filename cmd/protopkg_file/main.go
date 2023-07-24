package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	pppb "github.com/stackb/apis/build/stack/protobuf/package/v1alpha2"
	"github.com/stackb/protoreflecthash"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type flagName string

const (
	protoCompilerNameFlagName               flagName = "proto_compiler_name"
	protoCompilerVersionFileFlagName        flagName = "proto_compiler_version_file"
	protoDescriptorSetFileFlagName          flagName = "proto_descriptor_set_file"
	protoSourceFilesFlagName                flagName = "proto_source_files"
	protoRepositoryHostFlagName             flagName = "proto_repository_host"
	protoRepositoryOwnerFlagName            flagName = "proto_repository_owner"
	protoRepositoryRepoFlagName             flagName = "proto_repository_repo"
	protoRepositoryCommitFlagName           flagName = "proto_repository_commit"
	protoRepositoryRootFlagName             flagName = "proto_repository_root"
	protoFileDirectDependenciesFileFlagName flagName = "proto_file_direct_dependency_files"
	protoOutputFileFlagName                 flagName = "proto_out"
	jsonOutputFileFlagName                  flagName = "json_out"
)

var (
	protoCompilerName                    = flag.String(string(protoCompilerNameFlagName), "", "proto compiler name")
	protoCompilerVersionFile             = flag.String(string(protoCompilerVersionFileFlagName), "", "path to the proto_compiler version file")
	protoSourceFiles                     = flag.String(string(protoSourceFilesFlagName), "", "comma-separated path list path to the proto source files")
	protoDescriptorSetFile               = flag.String(string(protoDescriptorSetFileFlagName), "", "path to the compiled FileDescriptoSet")
	protoPackageSetDirectDependencyFiles = flag.String(string(protoFileDirectDependenciesFileFlagName), "", "comma-separated path list to a proto packages that represents the direct package dependencies of this one")
	protoRepositoryHost                  = flag.String(string(protoRepositoryHostFlagName), "", "value of the proto_repository.host")
	protoRepositoryOwner                 = flag.String(string(protoRepositoryOwnerFlagName), "", "value of the proto_repository.owner")
	protoRepositoryRepo                  = flag.String(string(protoRepositoryRepoFlagName), "", "value of the proto_repository.repo")
	protoRepositoryCommit                = flag.String(string(protoRepositoryCommitFlagName), "", "value of the proto_repository.commit")
	protoRepositoryRoot                  = flag.String(string(protoRepositoryRootFlagName), "", "value of the proto_repository.root")
	protoOutputFile                      = flag.String(string(protoOutputFileFlagName), "", "path of file to write the generated proto file")
	jsonOutputFile                       = flag.String(string(jsonOutputFileFlagName), "", "path of file to write the generated json file")
)

var (
	fileDeps    = make(map[string]*pppb.ProtoFile)
	packageDeps = make(map[string]*pppb.ProtoPackage)
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
	}
}

func run() error {
	flag.Parse()

	deps, err := readProtoPackageSetDirectDependencies(protoFileDirectDependenciesFileFlagName, *protoPackageSetDirectDependencyFiles)
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

	location, err := makeProtoArchive()
	if err != nil {
		return err
	}

	sourceMap, err := readProtoSourceFiles(protoSourceFilesFlagName, *protoSourceFiles)
	if err != nil {
		return err
	}

	pkg, err := makeProtoPackage(protoDescriptorSetData, protoDescriptorSet, location, compiler, sourceMap)
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

func makeProtoArchive() (*pppb.ProtoArchive, error) {
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
	archive := &pppb.ProtoArchive{
		Repository: &pppb.ProtoRepository{
			Host:     *protoRepositoryHost,
			Owner:    *protoRepositoryOwner,
			Name:     *protoRepositoryRepo,
			FullName: fmt.Sprintf("%s/%s/%s", *protoRepositoryHost, *protoRepositoryOwner, *protoRepositoryRepo),
		},
		CommitSha1: *protoRepositoryCommit,
		Root:       *protoRepositoryRoot,
	}
	archive.ShortSha1 = archive.CommitSha1[0:7]

	if err := collectArchiveCommitDetails(archive); err != nil {
		return nil, err
	}

	return archive, nil
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

func readProtoSourceFiles(flag flagName, commaSeparatedfilenames string) (map[string][]byte, error) {
	sourceMap := make(map[string][]byte)

	if commaSeparatedfilenames != "" {
		filenames := strings.Split(commaSeparatedfilenames, ",")
		for _, filename := range filenames {
			data, err := readProtoSourceFile(flag, filename)
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", filename, err)
			}
			sourceMap[filename] = data
		}
	}

	return sourceMap, nil
}

func readProtoSourceFile(flag flagName, filename string) ([]byte, error) {
	if filename == "" {
		return nil, errorFlagRequired(flag)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", flag, err)
	}
	return data, nil
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
	archive *pppb.ProtoArchive,
	compiler *pppb.ProtoCompiler,
	sourceMap map[string][]byte,
) (*pppb.ProtoPackage, error) {

	protoFiles := make([]*pppb.ProtoFile, len(ds.File))
	for i, file := range ds.File {
		protoFile, err := makeProtoFile(file)
		if err != nil {
			return nil, fmt.Errorf("making ProtoFile %d %s: %w", i, *file.Name, err)
		}
		var sourceCode string
		for filename, data := range sourceMap {
			if strings.HasSuffix(filename, *file.Name) {
				sourceCode = string(data)
				delete(sourceMap, filename)
				break
			}
		}
		if sourceCode == "" {
			return nil, fmt.Errorf("failed to collect source code for %q", *file.Name)
		}
		protoFile.SourceCode = sourceCode
		protoFiles[i] = protoFile
	}

	hash, err := makeProtoPackageHash(protoFiles)
	if err != nil {
		return nil, fmt.Errorf("calculating proto package hash: %w", err)
	}

	pkg := &pppb.ProtoPackage{
		Archive:      archive,
		Compiler:     compiler,
		Files:        protoFiles,
		Hash:         hash,
		Dependencies: makeProtoPackageDependencies(),
	}

	pkg.Name = makeProtoPackageName(pkg)

	return pkg, nil
}

func makeProtoPackageName(pkg *pppb.ProtoPackage) string {
	name := path.Join(pkg.Archive.Repository.FullName, pkg.Archive.Root)
	if len(pkg.Files) == 1 {
		name = name + ":" + *pkg.Files[0].File.Name
	} else {
		name = name + ":" + pkg.Hash
	}
	return name + "@" + pkg.Archive.CommitSha1
}

func makeProtoFile(file *descriptorpb.FileDescriptorProto) (*pppb.ProtoFile, error) {
	sortFile(file)

	data, err := proto.Marshal(file)
	if err != nil {
		return nil, fmt.Errorf("marshaling asset FileDescriptorProto: %w", err)
	}
	hash, err := protoreflectHash(file)
	if err != nil {
		return nil, fmt.Errorf("calculating fileset hash: %w", err)
	}
	deps, err := makeProtoFileDependencies(file.Dependency)
	if err != nil {
		return nil, fmt.Errorf("assembling file deps: %w", err)
	}

	return &pppb.ProtoFile{
		File:         file,
		FileSha256:   sha256Bytes(data),
		FileSize:     int64(len(data)),
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

func makeProtoFileDependencies(deps []string) ([]string, error) {
	results := make([]string, len(deps))
	for i, dep := range deps {
		file, ok := fileDeps[dep]
		if !ok {
			names := make([]string, 0, len(fileDeps))
			for name := range fileDeps {
				names = append(names, name)
			}
			log.Printf("file dependency not found for: %s (must be one of %v)", dep, names)
			results[i] = "DEP NOT FOUND: " + dep
			continue
			// return nil, fmt.Errorf("file dependency not found for: %s", dep)
		}
		results[i] = fileHashKey(file)
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
		for _, file := range pkg.Files {
			fileDeps[*file.File.Name] = file
		}
	}
}

func packageHashKey(pkg *pppb.ProtoPackage) string {
	return fmt.Sprintf("%s/%s@%s",
		pkg.Archive.Repository.FullName,
		makePackagePrefix(pkg.Archive.Root),
		pkg.Hash,
	)
}

func fileHashKey(file *pppb.ProtoFile) string {
	return fmt.Sprintf("%s@%s", *file.File.Name, file.Hash)
}

func makeProtoPackageHash(files []*pppb.ProtoFile) (string, error) {
	return protoreflectHash(&pppb.ProtoPackage{
		Files: files,
	})
}

func makePackagePrefix(prefix string) string {
	if prefix == "" {
		prefix = "~"
	}
	return prefix
}

func collectArchiveCommitDetails(archive *pppb.ProtoArchive) error {
	ghc := createGithubClient()
	ctx := context.Background()
	commit, _, err := ghc.Git.GetCommit(ctx, archive.Repository.Owner, archive.Repository.Name, archive.CommitSha1)
	if err != nil {
		return fmt.Errorf("gathering git commit details: %v", err)
	}
	archive.CommitMessage = commit.GetMessage()
	archive.CommitAuthor = commit.Author.GetEmail()
	archive.CommitTime = timestamppb.New(commit.GetAuthor().GetDate())
	return nil
}

// Create new client.
func createGithubClient() *github.Client {
	username := os.Getenv("GITHUB_USER")
	password := os.Getenv("GITHUB_TOKEN")
	cacheDir := os.Getenv("GITHUB_CACHE_DIR")

	// Create a BasicAuthTransport if the user has these env var
	// configured
	var basicAuth *github.BasicAuthTransport
	if username != "" && password != "" {
		basicAuth = &github.BasicAuthTransport{
			Username: strings.TrimSpace(username),
			Password: strings.TrimSpace(password),
		}
	}

	// Create a cache/transport implementation
	var cacheTransport *httpcache.Transport
	cache := diskcache.New(cacheDir)
	cacheTransport = httpcache.NewTransport(cache)

	// Create a Client
	if basicAuth != nil {
		basicAuth.Transport = cacheTransport
		return github.NewClient(basicAuth.Client())
	}

	return github.NewClient(cacheTransport.Client())
}
