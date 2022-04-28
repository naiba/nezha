package singleton

import (
	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var Localizer *i18n.Localizer

func InitLocalizer() {
	bundle := i18n.NewBundle(language.Chinese)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	_, err := bundle.LoadMessageFile("resource/l10n/" + Conf.Language + ".toml")
	if err != nil {
		panic(err)
	}
	_, err = bundle.LoadMessageFile("resource/l10n/zh-CN.toml")
	if err != nil {
		panic(err)
	}
	Localizer = i18n.NewLocalizer(bundle, Conf.Language)
}
