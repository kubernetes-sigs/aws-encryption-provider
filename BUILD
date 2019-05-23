#  Copyright 2018 The Kubernetes Authors.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

# gazelle:proto disable
load("@bazel_gazelle//:def.bzl", "gazelle")
load("@io_kubernetes_build//defs:run_in_workspace.bzl", "workspace_binary")

gazelle(
    name = "gazelle",
    external = "vendored",
    prefix = "github.com/kubernetes-sigs/aws-encryption-provider",
)

workspace_binary(
    name = "lint",
    args = ["run"],
    cmd = "@com_github_golangci_golangci-lint//cmd/golangci-lint",
)

workspace_binary(
    name = "lint-full",
    args = ["run --fast=false"],
    cmd = "@com_github_golangci_golangci-lint//cmd/golangci-lint",
)
