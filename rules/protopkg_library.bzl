load("@build_stack_rules_proto//rules:providers.bzl", "ProtoRepositoryInfo")
load("//rules:providers.bzl", "ProtoCompilerInfo", "ProtoPackageInfo")

def _protopkg_library_impl(ctx):
    proto_repository_info = ctx.attr.proto_repository[ProtoRepositoryInfo]
    proto_compiler_info = ctx.attr.proto_compiler[ProtoCompilerInfo]
    proto_info = ctx.attr.proto[ProtoInfo]
    proto_descriptor_set_file = proto_info.direct_descriptor_set
    proto_compiler_version_file = proto_compiler_info.version_file

    args = ctx.actions.args()
    args.add("-proto_descriptor_set_file", proto_descriptor_set_file.path)
    args.add("-proto_repository_host", proto_repository_info.source_host)
    args.add("-proto_repository_owner", proto_repository_info.source_owner)
    args.add("-proto_repository_repo", proto_repository_info.source_repo)
    args.add("-proto_repository_commit", proto_repository_info.source_commit)
    args.add("-proto_repository_prefix", proto_repository_info.source_prefix)
    args.add("-proto_repository_commit", proto_repository_info.source_commit)
    args.add("-proto_compiler_name", proto_compiler_info.name)
    args.add("-proto_compiler_version_file", proto_compiler_version_file.path)

    inputs = [
        proto_descriptor_set_file,
        proto_compiler_version_file,
    ]

    ctx.actions.run(
        executable = ctx.executable._tool,
        arguments = [args] + ["-proto_out", ctx.outputs.proto.path],
        inputs = inputs,
        outputs = [ctx.outputs.proto],
    )

    ctx.actions.run(
        executable = ctx.executable._tool,
        arguments = [args] + ["-json_out", ctx.outputs.json.path],
        inputs = inputs,
        outputs = [ctx.outputs.json],
    )

    return [
        DefaultInfo(
            files = depset([ctx.outputs.proto]),
        ),
        OutputGroupInfo(
            json = depset([ctx.outputs.json]),
        ),
        ProtoPackageInfo(
            label = ctx.label,
            proto_package_file = ctx.outputs.proto,
            proto_info = proto_info,
        ),
    ]

_protopkg_library = rule(
    implementation = _protopkg_library_impl,
    attrs = {
        "proto": attr.label(
            doc = "proto_library dependency",
            mandatory = True,
            providers = [ProtoInfo],
        ),
        "deps": attr.label_list(
            doc = "protopkg_library dependencies",
            providers = [ProtoPackageInfo],
        ),
        "proto_repository": attr.label(
            mandatory = True,
            providers = [ProtoRepositoryInfo],
        ),
        "proto_compiler": attr.label(
            mandatory = True,
            providers = [ProtoCompilerInfo],
        ),
        "_tool": attr.label(
            default = str(Label("//cmd/protopkg_library")),
            executable = True,
            cfg = "exec",
        ),
    },
    outputs = {
        "proto": "%{name}.pkg.pb",
        "json": "%{name}.pkg.json",
    },
)

def _protopkg_create_impl(ctx):
    pkg = ctx.attr.pkg[ProtoPackageInfo]

    script = """
#/bin/bash
set -euo pipefail

{executable} \
    -proto_package_file={file} \
    -packages_server_address={address}

    """.format(
        executable = ctx.executable._protopkg_create.short_path,
        file = pkg.proto_package_file.short_path,
        address = ctx.attr.address,
    )

    ctx.actions.write(
        ctx.outputs.executable,
        script,
        is_executable = True,
    )

    runfiles = ctx.runfiles(
        files = [
            ctx.executable._protopkg_create,
            pkg.proto_package_file,
        ],
        collect_data = True,
        collect_default = True,
    )

    return [DefaultInfo(
        files = depset([ctx.outputs.executable]),
        runfiles = runfiles,
        executable = ctx.outputs.executable,
    )]

_protopkg_create = rule(
    implementation = _protopkg_create_impl,
    attrs = {
        "pkg": attr.label(
            doc = "protopkg_library dependency",
            mandatory = True,
            providers = [ProtoPackageInfo],
        ),
        "address": attr.string(
            default = "localhost:4500",
        ),
        "_protopkg_create": attr.label(
            default = str(Label("//cmd/protopkg_create")),
            executable = True,
            cfg = "exec",
        ),
    },
    executable = True,
)

def protopkg_library(**kwargs):
    name = kwargs.pop("name")

    _protopkg_library(name = name, **kwargs)

    _protopkg_create(
        name = name + ".create",
        pkg = name,
    )
