package main

import (
	"fmt"

	"github.com/naiba/nezha/service/singleton"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func htmlTemplateTranslateFn(id string, data interface{}, count interface{}) string {
	return singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID:    id,
		TemplateData: data,
		PluralCount:  count,
	})
}

func main() {
	singleton.InitConfigFromPath("data/config.yaml")
	singleton.InitLocalizer()
	fmt.Println(singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "nezhaMonitor",
	}))

	fmt.Println(singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "nezhaMonitor",
	}))

	fmt.Println("tr nezhaMonitor", htmlTemplateTranslateFn("nezhaMonitor", nil, nil))
	fmt.Println("tr nezhaMonitor", htmlTemplateTranslateFn("nezhaMonitor", nil, 2))
	fmt.Println("tr nezhaMonitor", htmlTemplateTranslateFn("nezhaMonitor", map[string]string{
		"Ext": "Plus",
	}, 2))

	bundle := i18n.NewBundle(language.English)
	localizer := i18n.NewLocalizer(bundle, "en")
	catsMessage := &i18n.Message{
		ID:    "Cats",
		One:   "I have {{.PluralCount}} cat.",
		Other: "I have {{.PluralCount}} cats.",
	}
	fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: catsMessage,
		PluralCount:    1,
	}))
	fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: catsMessage,
		PluralCount:    2,
	}))
}
