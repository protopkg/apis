"""starlark rules definitions"""

def _make_protopkg_package_rule(rctx, pctx):
    r = gazelle.Rule(
        kind = "protopkg_package",
        name = pctx.proto_library.base_name + "_package",
        attrs = {
            "proto": pctx.proto_library.name,
            "deps": [],
            "proto_compiler": "@//:proto_compiler",
            "proto_repository": "//:proto_repository",
            "visibility": rctx.visibility,
        },
    )
    return r

def _provide_protopkg_package(rctx, pctx):
    return struct(
        name = "protopkg_package",
        rule = lambda: _make_protopkg_package_rule(rctx, pctx),
        experimental_resolve_attr = "deps",
    )

protoc.Rule(
    name = "protopkg_package",
    load_info = lambda: gazelle.LoadInfo(name = "@com_github_protopkg_protoregistry//rules:protopkg_package.bzl", symbols = ["protopkg_package"]),
    kind_info = lambda: gazelle.KindInfo(resolve_attrs = {"deps": True}),
    provide_rule = _provide_protopkg_package,
)

def _configure_protopkg_package(ctx):
    """_configure_protopkg_package prepares the PluginConfiguration for a fictitious protoc plugin.

    The purpose for this plugin definition is to ensure at least one output file is "predicted"
    foreach proto_library rule.  This produces a 1:1 correlation for protopkg_package deps.

    Args:
        ctx (protoc.PluginContext): The context object.
    Returns:
        config (PluginConfiguration): The configured PluginConfiguration object.
    """

    pb = ctx.proto_library.base_name + ".protopkg.pb"
    if ctx.rel:
        pb = "/".join([ctx.rel, pb])

    config = protoc.PluginConfiguration(
        label = "@//plugin:protoc-gen-protopkg",
        outputs = [pb],
        out = pb,
        options = ctx.plugin_config.options,
    )

    return config

protoc.Plugin(
    name = "protopkg_package_plugin",
    configure = _configure_protopkg_package,
)
