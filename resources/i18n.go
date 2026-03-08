package resources

import (
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/yaml.v3"
)

//go:embed locales/*.yaml
var localesFS embed.FS

var (
	defaultLang  = language.English
	printers     = make(map[language.Tag]*message.Printer)
	translations = make(map[language.Tag]map[string]any)
)

func init() {
	if err := Load(); err != nil {
		panic(fmt.Sprintf("failed to load translations: %v", err))
	}
}

// Load loads all translation files
func Load() error {
	files := map[language.Tag]string{
		language.English: "locales/en.i18n.yaml",
		language.German:  "locales/de.i18n.yaml",
		language.French:  "locales/fr.i18n.yaml",
		language.Spanish: "locales/es.i18n.yaml",
	}
	for langTag, file := range files {
		// Load translations from embedded files
		data, err := localesFS.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read translation file: %w", err)
		}
		var langTranslations map[string]any
		if err := yaml.Unmarshal(data, &langTranslations); err != nil {
			return fmt.Errorf("failed to unmarshal translations: %w", err)
		}
		translations[langTag] = langTranslations
	}

	return nil
}

// GetPrinter returns a message printer for the given language
func GetPrinter(lang language.Tag) *message.Printer {
	if p, ok := printers[lang]; ok {
		return p
	}
	p := message.NewPrinter(lang)
	printers[lang] = p
	return p
}

// T translates a key with optional template data
func T(key string, data ...map[string]interface{}) string {
	return Translate(defaultLang, key, data...)
}

// Translate translates a key for a specific language with optional template data
func Translate(lang language.Tag, key string, data ...map[string]interface{}) string {
	// Navigate through the nested map structure
	keys := strings.Split(key, ".")
	var current interface{} = translations[lang]

	for _, k := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[k]
		} else {
			return key // Return the key if not found
		}
	}

	// Convert to string
	str, ok := current.(string)
	if !ok {
		return key
	}

	// If template data is provided, apply it
	if len(data) > 0 && len(data[0]) > 0 {
		tmpl, err := template.New("trans").Parse(str)
		if err != nil {
			return str
		}

		var buf strings.Builder
		if err := tmpl.Execute(&buf, data[0]); err != nil {
			return str
		}
		return buf.String()
	}

	return str
}

// TData is a helper to create template data maps
func TData(pairs ...interface{}) map[string]interface{} {
	if len(pairs)%2 != 0 {
		panic("TData requires an even number of arguments")
	}

	data := make(map[string]interface{}, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			panic("TData keys must be strings")
		}
		data[key] = pairs[i+1]
	}
	return data
}

// GetLocalizations returns a map of Discord locale codes to translations for a given key
func GetLocalizations(key string) *map[discordgo.Locale]string {
	localizations := make(map[discordgo.Locale]string)

	localeMap := map[language.Tag]discordgo.Locale{
		language.German:  discordgo.German,
		language.French:  discordgo.French,
		language.Spanish: discordgo.SpanishES,
	}

	for lang, locale := range localeMap {
		if translated := Translate(lang, key); translated != key {
			localizations[locale] = translated
		}
	}

	if len(localizations) == 0 {
		return nil
	}

	return &localizations
}

// LocaleToLanguage converts a Discord locale to a language.Tag
func LocaleToLanguage(locale discordgo.Locale) language.Tag {
	switch locale {
	case discordgo.German:
		return language.German
	case discordgo.French:
		return language.French
	case discordgo.SpanishES:
		return language.Spanish
	case discordgo.EnglishUS, discordgo.EnglishGB:
		return language.English
	default:
		return defaultLang
	}
}

// TL translates a key for a Discord locale with optional template data
func TL(locale discordgo.Locale, key string, data ...map[string]interface{}) string {
	lang := LocaleToLanguage(locale)
	return Translate(lang, key, data...)
}
