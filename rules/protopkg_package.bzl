load("//rules:providers.bzl", "ProtoFileInfo", "ProtoPackageInfo")

def _protopkg_package_impl(ctx):
    deps = [dep[ProtoFileInfo] for dep in ctx.attr.deps]
    deps_files = depset([info.output_file for info in deps])

    args = ctx.actions.args()
    for dep_file in deps_files.to_list():
        args.add("-dep", dep_file.path)

    inputs = deps_files.to_list()

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
            output_file = ctx.outputs.proto,
            deps = deps,
        ),
    ]

_protopkg_package = rule(
    implementation = _protopkg_package_impl,
    attrs = {
        "deps": attr.label_list(
            doc = "protopkg_file dependencies",
            providers = [ProtoFileInfo],
        ),
        "_tool": attr.label(
            default = str(Label("//cmd/protopkg_package")),
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
    -output_file={file} \
    -packages_server_address={address}

    """.format(
        executable = ctx.executable._protopkg_create.short_path,
        file = pkg.output_file.short_path,
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
            pkg.output_file,
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
            doc = "protopkg_package dependency",
            mandatory = True,
            providers = [ProtoPackageInfo],
        ),
        "address": attr.string(
            default = "localhost:1080",
        ),
        "_protopkg_create": attr.label(
            default = str(Label("//cmd/protopkg_create")),
            executable = True,
            cfg = "exec",
        ),
    },
    executable = True,
)

def protopkg_package(**kwargs):
    name = kwargs.pop("name")

    _protopkg_package(name = name, **kwargs)

    _protopkg_create(
        name = name + ".create",
        pkg = name,
    )
