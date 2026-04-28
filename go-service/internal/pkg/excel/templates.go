package excel

import "embed"

//go:embed templates/*.xlsx
var TemplateFS embed.FS
