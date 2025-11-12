//go:build generate

package filters

//go:generate rm -f ./zz_*
//go:generate go run ../cmd/generator/. --debug --output-file=zz_provider-upjet-aws.go --repository-dir=../_work/providers/provider-upjet-aws --repo-url="https://github.com/crossplane-contrib/provider-upjet-aws.git"
//go:generate go fmt ./...
