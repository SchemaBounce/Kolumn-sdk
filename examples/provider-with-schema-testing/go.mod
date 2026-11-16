module github.com/schemabounce/kolumn/sdk/examples/provider-with-schema-testing

go 1.22.0

require (
	github.com/schemabounce/kolumn/sdk v0.0.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Build and test against the SDK in this repository, not a published version.
replace github.com/schemabounce/kolumn/sdk => ../..
