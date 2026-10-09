package migrations

import "embed"

// FS содержит встроенные файлы SQL-миграций схемы Gateway
//
//go:embed *.sql
var FS embed.FS
