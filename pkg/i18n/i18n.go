package i18n

import (
	"embed"
	"fmt"
	"sync"

	"github.com/chai2010/gettext-go"
)

//go:embed translations
var Translations embed.FS

var Languages = map[string]string{
	"zh_CN": "简体中文",
	"zh_TW": "繁體中文",
	"en_US": "English",
	"es_ES": "Español",
	"de_DE": "Deutsch",
}

type Localizer struct {
	intlMap map[string]gettext.Gettexter
	lang    string

	mu sync.RWMutex
}

func NewLocalizer(lang, domain, path string, data any) *Localizer {
	intl := gettext.New(domain, path, data)
	intl.SetLanguage(lang)

	intlMap := make(map[string]gettext.Gettexter)
	intlMap[lang] = intl

	return &Localizer{intlMap: intlMap, lang: lang}
}

func (l *Localizer) SetLanguage(lang string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.lang = lang
}

func (l *Localizer) Exists(lang string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if _, ok := l.intlMap[lang]; ok {
		return ok
	}
	return false
}

func (l *Localizer) AppendIntl(lang, domain, path string, data any) {
	intl := gettext.New(domain, path, data)
	intl.SetLanguage(lang)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.intlMap[lang] = intl
}

// Modified from k8s.io/kubectl/pkg/util/i18n

func (l *Localizer) T(orig string) string {
	l.mu.RLock()
	intl, ok := l.intlMap[l.lang]
	l.mu.RUnlock()
	if !ok {
		return orig
	}

	return intl.PGettext("", orig)
}

// N translates a string, possibly substituting arguments into it along
// the way. If len(args) is > 0, args1 is assumed to be the plural value
// and plural translation is used.
func (l *Localizer) N(orig string, args ...int) string {
	l.mu.RLock()
	intl, ok := l.intlMap[l.lang]
	l.mu.RUnlock()
	if !ok {
		return orig
	}

	if len(args) == 0 {
		return intl.PGettext("", orig)
	}
	return fmt.Sprintf(intl.PNGettext("", orig, orig+".plural", args[0]),
		args[0])
}

// ErrorT produces an error with a translated error string.
// Substitution is performed via the `T` function above, following
// the same rules.
func (l *Localizer) ErrorT(defaultValue string, args ...any) error {
	return fmt.Errorf(l.T(defaultValue), args...)
}

func (l *Localizer) Tf(defaultValue string, args ...any) string {
	return fmt.Sprintf(l.T(defaultValue), args...)
}
