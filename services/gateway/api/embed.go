package api

import (
	_ "embed"
)

// OpenAPIYAML contains the raw embedded OpenAPI 3.0 specification for Errorium Gateway.
//
//go:embed openapi.yaml
var OpenAPIYAML []byte
