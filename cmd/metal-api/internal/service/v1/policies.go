package v1

import "embed"

//go:embed policies/*.rego
var RegoPolicies embed.FS
