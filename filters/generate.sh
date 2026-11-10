#!/bin/sh
# Regenerate the resource filters from the provider versions pinned in
# versions.env. Used by `go generate` and .github/workflows/update-filters.yml.
#
# Each provider version is cloned into its own directory under _work, so a
# changed pin always clones the new tag and an existing clone is only reused
# for the same version.
set -eu

cd "$(dirname "$0")"

. ./versions.env

generate() {
	provider=$1
	version=$2
	template=$3

	go run ../cmd/generator/. --debug \
		--output-file="zz_${provider}.go" \
		--repository-dir="../_work/providers/${provider}-${version}" \
		--repo-url="https://github.com/crossplane-contrib/${provider}.git" \
		--template-file="../templates/${template}.tmpl" \
		--git-branch-origin-main="refs/tags/${version}"
}

generate provider-upjet-aws "${UPJET_AWS_VERSION}" aws
generate provider-upjet-azure "${UPJET_AZURE_VERSION}" azure

go fmt ./...
