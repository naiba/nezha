package resource

import (
	"embed"

	"github.com/naiba/nezha/pkg/utils"
)

var StaticFS *utils.HybridFS

//go:embed static
var staticFS embed.FS

//go:embed template
var TemplateFS embed.FS

//go:embed l10n
var I18nFS embed.FS

func init() {
	var err error
	StaticFS, err = utils.NewHybridFS(staticFS, "static", "resource/static/custom")
	if err != nil {
		panic(err)
	}
}

func IsTemplateFileExist(name string) bool {
	_, err := TemplateFS.Open(name)
	return err == nil
}
