"""starlark rules definitions"""

def _make_proto_repository_info_rule(rctx, pctx):
    return gazelle.Rule(
        kind = "proto_repository_info",
        name = "proto_repository_info",
        attrs = {
            "srcs": ["proto_repository.info.json"],
            "visibility": rctx.visibility,
        },
    )

def _provide_proto_repository_info(rctx, pctx):
    return struct(
        name = "proto_repository_info",
        rule = lambda: _make_proto_repository_info_rule(rctx, pctx),
    )

protoc.Rule(
    name = "proto_repository_info",
    load_info = lambda: gazelle.LoadInfo(name = "@protopkg_apis//rules:proto_repository_info.bzl", symbols = ["proto_repository_info"]),
    kind_info = lambda: gazelle.KindInfo(merge_attrs = {"srcs": True}),
    provide_rule = _provide_proto_repository_info,
)
