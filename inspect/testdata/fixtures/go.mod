module example.com/testmodule

go 1.21

require (
	github.com/some/package v1.2.3
	github.com/another/package v0.2.0 // indirect
)

replace (
	github.com/some/package => ../some-package
	github.com/another/package v0.1.0 => github.com/new/package v0.3.0
)
