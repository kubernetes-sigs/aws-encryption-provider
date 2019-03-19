.PHONY: gofmt
gofmt:
	gofmt -w -s cmd/ pkg/ test/

.PHONY: goimports
goimports:
	goimports -w cmd/ pkg/ test/

.PHONY: lint
lint: dep-ensure ## Lint codebase
	bazel run //:lint $(BAZEL_ARGS)

.PHONY: lint-full
lint-full: dep-ensure ## Run slower linters to detect possible issues
	bazel run //:lint-full $(BAZEL_ARGS)

.PHONY: test
test:
	bazel test //pkg/... //test/... --test_output=streamed

.PHONY: push
push:
	bazel run //images:push-aws-encryption-provider

.PHONY: dep-ensure
dep-ensure:
	dep ensure -v
	bazel run //:gazelle
