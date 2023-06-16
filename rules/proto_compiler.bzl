load("//rules:providers.bzl", "ProtoCompilerInfo")

def _proto_compiler_impl(ctx):
    ctx.actions.run_shell(
        command = "{compiler} {arguments} > {version_file}".format(
            compiler = ctx.executable.compiler.path,
            arguments = " ".join(ctx.attr.arguments),
            version_file = ctx.outputs.version.path,
        ),
        tools = [ctx.executable.compiler],
        outputs = [ctx.outputs.version],
    )

    return [
        OutputGroupInfo(
            proto_compiler_version = depset([ctx.outputs.version]),
        ),
        ProtoCompilerInfo(
            name = ctx.attr.compiler_name,
            version_file = ctx.outputs.version,
        ),
    ]

proto_compiler = rule(
    implementation = _proto_compiler_impl,
    attrs = {
        "compiler_name": attr.string(
            default = "protoc",
        ),
        "arguments": attr.string_list(
            default = ["--version", "|", "sed", "'s/^libprotoc //'"],
        ),
        "compiler": attr.label(
            default = "@com_google_protobuf//:protoc",
            executable = True,
            cfg = "exec",
        ),
    },
    outputs = {
        "version": "%{name}.version",
    },
)
