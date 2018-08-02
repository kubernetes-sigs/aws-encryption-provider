.PHONY: gofmt
gofmt:
	gofmt -w -s cmd/ pkg/ test/

.PHONY: goimports
goimports:
	goimports -w cmd/ pkg/ test/

.PHONY: test
test:
	bazel test //pkg/... //test/... --test_output=streamed

.PHONY: push
push:
	bazel run //images:push-aws-encryption-provider

.PHONY: dep-ensure
dep-ensure:
	dep ensure -v
	find vendor/ -name "BUILD" -delete
	find vendor/ -name "BUILD.bazel" -delete
	bazel run //:gazelle
