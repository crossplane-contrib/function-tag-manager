//go:build generate

package filters

//go:generate rm -f ./zz_*
//go:generate go run ../cmd/generator/. --debug --output-file=zz_provider-upjet-aws.go --repository-dir=../_work/providers/provider-upjet-aws --repo-url=https://github.com/crossplane-contrib/provider-upjet-aws.git --template-file=../templates/aws.tmpl
//go:generate go run ../cmd/generator/. --debug --output-file=zz_provider-upjet-azure.go --repository-dir=../_work/providers/provider-upjet-azure --repo-url=https://github.com/crossplane-contrib/provider-upjet-azure.git --template-file=../templates/azure.tmpl
//go:generate go fmt ./...
