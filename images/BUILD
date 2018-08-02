package(default_visibility = ["//visibility:public"])

load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_image",
    "container_push",
)

container_image(
    name = "aws-encryption-provider",
    base = "@debian_base_amd64//image",
    entrypoint = ["/server"],
    files = [
        "//cmd/server",
    ],
)

container_push(
    name = "push-aws-encryption-provider",
    format = "Docker",
    image = ":aws-encryption-provider",
    registry = "{STABLE_DOCKER_REGISTRY}",
    repository = "{STABLE_DOCKER_IMAGE_PREFIX}aws-encryption-provider",
    stamp = True,
    tag = "{STABLE_DOCKER_TAG}",
)
