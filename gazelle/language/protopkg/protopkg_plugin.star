"""starlark plugin definitions"""

def _configure_protoc_gen_protopkg(ctx):
    """_configure_protoc_gen_protopkg prepares the PluginConfiguration for a fictitious protoc plugin.

    The purpose for this plugin definition is to ensure at least one output file is "predicted"
    foreach proto_library rule.  This produces a 1:1 correlation for protopkg_library deps.

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
    name = "protoc-gen-protopkg",
    configure = _configure_protoc_gen_protopkg,
)
