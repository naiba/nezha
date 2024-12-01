package singleton

import (
	"archive/zip"
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/nezhahq/nezha/pkg/i18n"
)

const domain = "nezha"

var Localizer *i18n.Localizer

func initI18n() {
	if err := loadTranslation(); err != nil {
		log.Printf("NEZHA>> init i18n failed: %v", err)
	}
}

func loadTranslation() error {
	lang := Conf.Language
	if lang == "" {
		lang = "zh_CN"
	}

	lang = strings.Replace(lang, "-", "_", 1)
	data, err := getTranslationArchive(lang)
	if err != nil {
		return err
	}

	Localizer = i18n.NewLocalizer(lang, domain, domain+".zip", data)
	return nil
}

func OnUpdateLang(lang string) error {
	lang = strings.Replace(lang, "-", "_", 1)
	if Localizer.Exists(lang) {
		Localizer.SetLanguage(lang)
		return nil
	}

	data, err := getTranslationArchive(lang)
	if err != nil {
		return err
	}

	Localizer.AppendIntl(lang, domain, domain+".zip", data)
	Localizer.SetLanguage(lang)
	return nil
}

func getTranslationArchive(lang string) ([]byte, error) {
	files := [...]string{
		fmt.Sprintf("translations/%s/LC_MESSAGES/%s.po", lang, domain),
		fmt.Sprintf("translations/%s/LC_MESSAGES/%s.mo", lang, domain),
	}

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for _, file := range files {
		f, err := w.Create(file)
		if err != nil {
			return nil, err
		}
		data, err := i18n.Translations.ReadFile(file)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(data); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
