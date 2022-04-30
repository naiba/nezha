package singleton

import (
	"log"

	"github.com/BurntSushi/toml"
	"github.com/naiba/nezha/pkg/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var Localizer *i18n.Localizer

func InitLocalizer() {
	bundle := i18n.NewBundle(language.Chinese)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	userCustomLanguageFile := "resource/l10n/" + Conf.Language + ".toml"

	if exists, err := utils.PathExists(userCustomLanguageFile); !exists {
		log.Println("NEZHA>> language file not found:", userCustomLanguageFile, err)
		Conf.Language = "zh-CN"
	} else {
		_, err := bundle.LoadMessageFile(userCustomLanguageFile)
		if err != nil {
			panic(err)
		}
	}

	if _, err := bundle.LoadMessageFile("resource/l10n/zh-CN.toml"); err != nil {
		panic(err)
	}
	Localizer = i18n.NewLocalizer(bundle, Conf.Language)
}
