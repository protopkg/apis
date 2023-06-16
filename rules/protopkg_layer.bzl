load("@bazel_skylib//lib:dicts.bzl", "dicts")
load("@io_bazel_rules_docker//container:providers.bzl", "LayerInfo")
load("@io_bazel_rules_docker//container:layer.bzl", "build_layer", "layer")
load("//rules:providers.bzl", "ProtoPackageInfo")

def _protopkg_layer_impl(ctx):
    pkg_infos = [dep[ProtoPackageInfo] for dep in ctx.attr.deps]

    layers = []
    for pkg_info in pkg_infos:
        files = []
        files.append(pkg_info.proto_package_file)
        files.append(pkg_info.proto_info.direct_descriptor_set)
        files.extend(pkg_info.proto_info.direct_sources)
        files.extend(pkg_info.proto_info.direct_sources)

        directory = pkg_info.label.package
        layer_name = "%s.%s.%s.layer.tar" % (pkg_info.label.workspace_name, pkg_info.label.package.replace("/", "-"), pkg_info.label.name)
        layer_file = ctx.actions.declare_file(layer_name)

        layer, sha256 = build_layer(
            ctx,
            layer_name,
            layer_file,
            files = files,
            directory = directory,
            file_map = {},
            symlinks = {},
            tars = [],
            debs = [],
        )
        layers.append(layer)

    return [
        DefaultInfo(files = depset(layers)),
        LayerInfo(),
    ]

protopkg_layer = rule(
    implementation = _protopkg_layer_impl,
    attrs = dicts.add({
        "deps": attr.label_list(
            doc = "protopkg_library dependencies",
            mandatory = True,
            providers = [ProtoPackageInfo],
        ),
    }, layer.attrs),
    toolchains = layer.toolchains,
)
