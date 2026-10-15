//go:build generate

package filters

// Regenerate the resource filters from the provider versions pinned in
// versions.env. Remove the _work directory first if you want a fresh clone;
// an existing clone is reused.
//go:generate rm -f ./zz_*
//go:generate sh -c '. ./versions.env && go run ../cmd/generator/. --debug --output-file=zz_provider-upjet-aws.go --repository-dir=../_work/providers/provider-upjet-aws --repo-url=https://github.com/crossplane-contrib/provider-upjet-aws.git --template-file=../templates/aws.tmpl --git-branch-origin-main=refs/tags/$UPJET_AWS_VERSION'
//go:generate sh -c '. ./versions.env && go run ../cmd/generator/. --debug --output-file=zz_provider-upjet-azure.go --repository-dir=../_work/providers/provider-upjet-azure --repo-url=https://github.com/crossplane-contrib/provider-upjet-azure.git --template-file=../templates/azure.tmpl --git-branch-origin-main=refs/tags/$UPJET_AZURE_VERSION'
//go:generate go fmt ./...
