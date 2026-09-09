MODULES := . clients/connectrpc_client clients/grpc_client launcher/ops/sentry \
	transport/connectrpc_transport transport/grpc_transport

.PHONY: fmt lint test tidy release-check release

fmt:
	go fix ./...
	gofumpt -l -w .

lint:
	@for m in $(MODULES); do echo "==> lint $$m"; (cd $$m && golangci-lint run ./...) || exit 1; done

test:
	@for m in $(MODULES); do echo "==> test $$m"; (cd $$m && go test -race -shuffle=on ./...) || exit 1; done

tidy:
	@for m in $(MODULES); do echo "==> tidy $$m"; (cd $$m && go mod tidy) || exit 1; done

# Release

# Validate the GoReleaser config without touching git or GitHub. Run it before
# `make release`: a broken config only surfaces after the tag is pushed.
release-check:
	goreleaser check

release: release-check
	@if [ -z "$(TAG)" ]; then echo "Usage: make release TAG=v1.2.3"; exit 1; fi
	@# GoReleaser refuses a tag it cannot read as semver, and by then the tag is
	@# already pushed, so reject it here instead. The pattern is semver's own
	@# grammar: a hyphen is legal inside a pre-release identifier and build
	@# metadata is legal after a plus, and GoReleaser accepts both.
	@printf '%s' "$(TAG)" | \
		grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$$' || \
		{ echo "TAG must be semver: v1.2.3, v1.2.3-rc.1, v1.2.3+build.7"; exit 1; }
	@git rev-parse -q --verify "refs/tags/$(TAG)" >/dev/null && \
		{ echo "tag $(TAG) already exists locally"; exit 1; } || true
	@# The release notes are built from the commits between the previous tag and
	@# this one, so what is tagged has to be reviewable afterwards: no uncommitted
	@# work in the build, and nothing that only exists in this clone.
	@git update-index -q --refresh
	@git diff-index --quiet HEAD -- || \
		{ echo "the working tree is dirty; commit or stash before releasing"; exit 1; }
	@git fetch --quiet origin master
	@git merge-base --is-ancestor HEAD FETCH_HEAD || \
		{ echo "HEAD is not on origin/master, which is the only branch CI gates"; exit 1; }
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin $(TAG)
