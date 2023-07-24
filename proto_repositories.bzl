load("@build_stack_rules_proto//rules/proto:proto_repository.bzl", "github_proto_repository")

def proto_repositories():
    github_proto_repository(
        name = "protoapis",
        owner = "protocolbuffers",
        repo = "protobuf",
        commit = "a74f54b724bdc2fe0bfc271f4dc0ceb159805625",
        prefix = "src",
        sha256 = "087c2ec84a07308318d35e0e39717e2037e05d14e628244602a2c78fbe203fa5",
        cfgs = ["//:rules_proto_config.yaml"],
        reresolve_known_proto_imports = True,
        build_directives = [
            "gazelle:exclude testdata",
            "gazelle:exclude google/protobuf/bridge",
            "gazelle:exclude google/protobuf/compiler/cpp",
            "gazelle:exclude google/protobuf/compiler/java",
            "gazelle:exclude google/protobuf/compiler/ruby",
            "gazelle:exclude google/protobuf/util",
            "gazelle:proto_language go enable true",
            "gazelle:proto_language protopkg enable true",
        ],
        deleted_files = [
            "google/protobuf/map_lite_unittest.proto",
            "google/protobuf/map_proto2_unittest.proto",
            "google/protobuf/map_proto3_unittest.proto",
            "google/protobuf/map_unittest.proto",
            "google/protobuf/test_messages_proto2.proto",
            "google/protobuf/test_messages_proto3.proto",
            "google/protobuf/any_test.proto",
            "google/protobuf/unittest.proto",
            "google/protobuf/unittest_arena.proto",
            "google/protobuf/unittest_custom_options.proto",
            "google/protobuf/unittest_drop_unknown_fields.proto",
            "google/protobuf/unittest_embed_optimize_for.proto",
            "google/protobuf/unittest_empty.proto",
            "google/protobuf/unittest_enormous_descriptor.proto",
            "google/protobuf/unittest_import_lite.proto",
            "google/protobuf/unittest_import_public_lite.proto",
            "google/protobuf/unittest_import_public.proto",
            "google/protobuf/unittest_import.proto",
            "google/protobuf/unittest_lazy_dependencies_custom_option.proto",
            "google/protobuf/unittest_lazy_dependencies_enum.proto",
            "google/protobuf/unittest_lazy_dependencies.proto",
            "google/protobuf/unittest_lite_imports_nonlite.proto",
            "google/protobuf/unittest_lite.proto",
            "google/protobuf/unittest_mset_wire_format.proto",
            "google/protobuf/unittest_mset.proto",
            "google/protobuf/unittest_no_field_presence.proto",
            "google/protobuf/unittest_no_generic_services.proto",
            "google/protobuf/unittest_optimize_for.proto",
            "google/protobuf/unittest_preserve_unknown_enum.proto",
            "google/protobuf/unittest_preserve_unknown_enum2.proto",
            "google/protobuf/unittest_proto3_arena_lite.proto",
            "google/protobuf/unittest_proto3_arena.proto",
            "google/protobuf/unittest_proto3_lite.proto",
            "google/protobuf/unittest_proto3_optional.proto",
            "google/protobuf/unittest_proto3.proto",
            "google/protobuf/unittest_retention.proto",
            "google/protobuf/unittest_well_known_types.proto",
            "google/protobuf/compiler/cpp/test_bad_identifiers.proto",
        ],
    )

    github_proto_repository(
        name = "googleapis",
        owner = "googleapis",
        repo = "googleapis",
        commit = "e115ab1839cb6e1bd953e40337b7e84001291766",
        sha256 = "e5b59ae2c0c812e3867158eca8e484fddb96dff03b8e2073bf44242b708fa919",
        reresolve_known_proto_imports = True,
        cfgs = ["//:rules_proto_config.yaml"],
        imports = ["@protoapis//:imports.csv"],
        build_directives = [
            "gazelle:exclude google/ads/googleads/v12/services",
            "gazelle:exclude google/ads/googleads/v13/services",
            "gazelle:exclude google/ads/googleads/v14/services",
            "gazelle:proto_language go enable true",
            "gazelle:proto_language protopkg enable true",
            "gazelle:proto_rule proto_compile attr args --experimental_allow_proto3_optional",
        ],
    )

    github_proto_repository(
        name = "grpcapis",
        owner = "grpc",
        repo = "grpc",
        commit = "3d9f2d8f77a65fe803035580a6f8786f0cb0db77",
        sha256 = "0b55e170c5f0f9bc1f963ac6deaec01ea438e77fca91a829f904a3bd2754564b",
        cfgs = ["//:rules_proto_config.yaml"],
        prefix = "src/proto",
        imports = ["@protoapis//:imports.csv"],
        build_directives = [
            "gazelle:exclude math",
            "gazelle:exclude grpc/gcp",
            "gazelle:exclude grpc/status",
            "gazelle:exclude grpc/testing",
            "gazelle:proto_language protopkg enable true",
            "gazelle:proto_rule proto_compile attr args --experimental_allow_proto3_optional",
        ],
    )

    # Commit: f7969e56f12bc9270e39e3099eeae514157ebddd
    # Date: 2023-07-23 21:57:15 +0000 UTC
    # URL: https://github.com/stackb/apis/commit/f7969e56f12bc9270e39e3099eeae514157ebddd
    #
    # Update generated files (#21)
    # Size: 52234 (52 kB)
    github_proto_repository(
        name = "stackbuildapis",
        owner = "stackb",
        repo = "apis",
        commit = "f7969e56f12bc9270e39e3099eeae514157ebddd",
        sha256 = "7e2724b75de59cf3afd793aed52a967683e56ede5dfa81722b3728fa682d9ba9",
        cfgs = ["//:rules_proto_config.yaml"],
        reresolve_known_proto_imports = True,
        imports = [
            "@protoapis//:imports.csv",
            "@googleapis//:imports.csv",
        ],
        build_directives = [
            "gazelle:proto_language go enable true",
            "gazelle:proto_language protopkg enable true",
        ],
    )

    github_proto_repository(
        name = "remoteapis",
        owner = "bazelbuild",
        repo = "remote-apis",
        commit = "cb8058798964f0adf6dbab2f4c2176ae2d653447",
        sha256 = "5d67e5aa65b7d95218714828fd647cece4be5949201c21353f5c2be1954b46e6",
        cfgs = ["//:rules_proto_config.yaml"],
        reresolve_known_proto_imports = True,
        imports = [
            "@protoapis//:imports.csv",
            "@googleapis//:imports.csv",
        ],
        build_directives = [
            "gazelle:exclude third_party",
            # "gazelle:exclude build/bazel/remote/asset/v1",
            # "gazelle:exclude build/bazel/remote/logstream/v1",
            "gazelle:proto_language protopkg enable true",
        ],
    )

    github_proto_repository(
        name = "bazelapis",
        owner = "bazelbuild",
        repo = "bazel",
        commit = "1da6f4f64180b4648bab167b602a1a878e9b5488",
        sha256 = "778bbf2cbbf367e090dd77a93dead3eaaf7c58fa2f6e6b0b1c159f294ba4e11a",
        cfgs = ["//:rules_proto_config.yaml"],
        reresolve_known_proto_imports = True,
        imports = [
            "@googleapis//:imports.csv",
            "@protoapis//:imports.csv",
            "@remoteapis//:imports.csv",
        ],
        build_directives = [
            "gazelle:exclude third_party",
            "gazelle:proto_language protopkg enable true",
        ],
    )
