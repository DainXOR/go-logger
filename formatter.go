package logger

import (
	"fmt"
	"strings"
	"time"

	"github.com/DainXOR/go-utils/datatypes"
	"github.com/DainXOR/go-utils/mapx"
	"github.com/DainXOR/go-utils/version"
)

// Formatter interface defines the methods required for custom formatter implementations.
// This interface also provides a guide on what chained formatters for a correct
// interaction between them.
type Formatter interface {
	Format(original *Record, formatRecord ...*FormatRecord) (string, error)
	Next() Formatter
	SetNext(next Formatter)
}

// FormatterBase provides a base implementation for the Formatter interface.
// It includes common formatting methods and a next formatter for chaining.
// You can use this as a base for your custom formatters to ensure that they
// implement the Formatter interface correctly.
// It is not needed if you want to implement a custom formatter.
//
// > Keep in mind the golang method overriding behavior
type FormatterBase struct {
	next       Formatter
	dateFormat string
}

// Returns the date format used to represent the time in the log record.
// Do not confuse with dateFormatString, which is used to "decorate" the time value
// in the log record.
func (f *FormatterBase) DateFormat() string {
	return f.dateFormat
}
func (f *FormatterBase) Next() Formatter {
	return f.next
}
func (f *FormatterBase) SetNext(next Formatter) {
	f.next = next
}
func (f *FormatterBase) FinalString(original *Record, formatRecord *FormatRecord) string {
	formattedLevel := fmt.Sprintf(formatRecord.LogLevel, original.LogLevel.Name())
	formattedTime := fmt.Sprintf(formatRecord.Time, original.Time.Format(f.DateFormat()))
	formattedFile := fmt.Sprintf(formatRecord.File, original.File)
	formattedLine := fmt.Sprintf(formatRecord.Line, fmt.Sprint(original.Line))
	formattedMessage := fmt.Sprintf(formatRecord.Message, original.Message)
	formattedVersion := fmt.Sprintf(formatRecord.AppVersion, original.AppVersion.String())

	formattedContext := ""
	if len(original.Context) != 0 {
		formattedContext = fmt.Sprintf(formatRecord.ContextBegin, "")
		for k, pair := range mapx.Zip(original.Context, formatRecord.Context, datatypes.NewSPair("%s:", " %s, ")) {
			formatKey, formatValue := pair.Second.First, pair.Second.Second
			formatStr := formatKey + formatValue
			value := pair.First

			formattedContext += fmt.Sprintf(formatStr, k, value)
		}
		formattedContext = strings.TrimSuffix(formattedContext, ", ")
		formattedContext += fmt.Sprintf(formatRecord.ContextEnd, "")
	}

	finalString := fmt.Sprint(
		formattedLevel,
		formattedTime,
		formattedFile,
		formattedLine,
		formattedMessage,
		formattedVersion,
		formattedContext,
	)

	return finalString
}

func (f *FormatterBase) DefaultFormat() string {
	return "%s"
}

func (f *FormatterBase) levelFormatString(_ logLevel) string {
	return f.DefaultFormat()
}
func (f *FormatterBase) dateFormatString(_ time.Time) string {
	return f.DefaultFormat()
}
func (f *FormatterBase) fileFormatString(_ string) string {
	return f.DefaultFormat()
}
func (f *FormatterBase) lineFormatString(_ int) string {
	return "%s"
}
func (f *FormatterBase) messageFormatString(_ string) string {
	return f.DefaultFormat()
}
func (f *FormatterBase) versionFormatString(_ version.Version) string {
	return f.DefaultFormat()
}
func (f *FormatterBase) contextFormatStrings(m map[string]string) map[string]datatypes.SPair[string] {
	formatMap := make(map[string]datatypes.SPair[string], len(m))
	for k := range m {
		formatMap[k] = datatypes.NewSPair(f.DefaultFormat(), f.DefaultFormat())
	}
	return formatMap
}
func (f *FormatterBase) contextPrefixString(_ map[string]string) string {
	return "%s"
}
func (f *FormatterBase) contextSuffixString(_ map[string]string) string {
	return "%s"
}

// Since go does not support method overriding the same way as other languages,
// if you "override" any of the methods in FormatterBase, you must also
// override the Format method to ensure that the correct methods are used.
// If you want to keep this behavior, you can simply copy this method
// and paste it in your custom formatter implementation, this will ensure that
// the methods called are the ones you defined in your custom formatter.
func (f *FormatterBase) Format(r *Record, formatRecord ...*FormatRecord) (string, error) {
	fr := &FormatRecord{}
	if len(formatRecord) > 0 {
		fr = formatRecord[0]
	}

	if f.Next() != nil {
		if _, err := f.Next().Format(r, fr); err != nil {
			return "", err
		}

		fr.LogLevel = fmt.Sprintf(fr.LogLevel, f.levelFormatString(r.LogLevel))
		fr.Time = fmt.Sprintf(fr.Time, f.dateFormatString(r.Time))
		fr.File = fmt.Sprintf(fr.File, f.fileFormatString(r.File))
		fr.Line = fmt.Sprintf(fr.Line, f.lineFormatString(r.Line))
		fr.Message = fmt.Sprintf(fr.Message, f.messageFormatString(r.Message))
		fr.AppVersion = fmt.Sprintf(fr.AppVersion, f.versionFormatString(r.AppVersion))
		fr.Context = mapx.Apply(fr.Context, func(k string, v datatypes.SPair[string]) datatypes.SPair[string] {
			last := f.contextFormatStrings(r.Context)[k]
			current := fr.Context[k]

			formatKey := fmt.Sprintf(current.First, last.First)
			formatValue := fmt.Sprintf(current.Second, last.Second)

			return datatypes.NewSPair(formatKey, formatValue)
		})
		fr.ContextBegin = fmt.Sprintf(fr.ContextBegin, f.contextPrefixString(r.Context))
		fr.ContextEnd = fmt.Sprintf(fr.ContextEnd, f.contextSuffixString(r.Context))
	} else {
		*fr = FormatRecord{
			LogLevel:     f.levelFormatString(r.LogLevel),
			Time:         f.dateFormatString(r.Time),
			File:         f.fileFormatString(r.File),
			Line:         f.lineFormatString(r.Line),
			Message:      f.messageFormatString(r.Message),
			AppVersion:   f.versionFormatString(r.AppVersion),
			Context:      f.contextFormatStrings(r.Context),
			ContextBegin: f.contextPrefixString(r.Context),
			ContextEnd:   f.contextSuffixString(r.Context),
		}
	}

	return f.FinalString(r, fr), nil
}

type FormatterBuilder interface {
	Next(Formatter) FormatterBuilder
	New() Formatter
}

/* SimpleFormatter implements a basic text formatter for log records.
 * It formats the log record into a string with a specific structure.
 * The default date format is "02/01/2006 15:04:05 -07:00".
 * You can customize the date format using the TimeFormat method.
 */
type simpleFormatter struct {
	FormatterBase
}

func (f *simpleFormatter) levelFormatString(_ logLevel) string {
	return "|%s| "
}
func (f *simpleFormatter) dateFormatString(_ time.Time) string {
	return "%s "
}
func (f *simpleFormatter) fileFormatString(_ string) string {
	return "%s:"
}
func (f *simpleFormatter) lineFormatString(_ int) string {
	return "%s:"
}
func (f *simpleFormatter) messageFormatString(_ string) string {
	return " %s"
}
func (f *simpleFormatter) versionFormatString(_ version.Version) string {
	return " [%s]"
}
func (f *simpleFormatter) contextFormatStrings(m map[string]string) map[string]datatypes.SPair[string] {
	formatMap := make(map[string]datatypes.SPair[string], len(m))
	for k := range m {
		formatMap[k] = datatypes.NewSPair("%s:", " %s, ") // "%s: %s, "
	}
	return formatMap
}
func (f *simpleFormatter) contextPrefixString(_ map[string]string) string {
	return " {%s"
}
func (f *simpleFormatter) contextSuffixString(_ map[string]string) string {
	return "%s}"
}

func (f *simpleFormatter) Format(r *Record, formatRecord ...*FormatRecord) (string, error) {
	fr := &FormatRecord{}
	if len(formatRecord) > 0 {
		fr = formatRecord[0]
	}

	if f.Next() != nil {
		if _, err := f.Next().Format(r, fr); err != nil {
			return "", err
		}

		fr.LogLevel = fmt.Sprintf(fr.LogLevel, f.levelFormatString(r.LogLevel))
		fr.Time = fmt.Sprintf(fr.Time, f.dateFormatString(r.Time))
		fr.File = fmt.Sprintf(fr.File, f.fileFormatString(r.File))
		fr.Line = fmt.Sprintf(fr.Line, f.lineFormatString(r.Line))
		fr.Message = fmt.Sprintf(fr.Message, f.messageFormatString(r.Message))
		fr.AppVersion = fmt.Sprintf(fr.AppVersion, f.versionFormatString(r.AppVersion))
		fr.Context = mapx.Apply(fr.Context, func(k string, v datatypes.SPair[string]) datatypes.SPair[string] {
			last := f.contextFormatStrings(r.Context)[k]
			current := fr.Context[k]

			formatKey := fmt.Sprintf(current.First, last.First)
			formatValue := fmt.Sprintf(current.Second, last.Second)

			return datatypes.NewSPair(formatKey, formatValue)
		})
		fr.ContextBegin = fmt.Sprintf(fr.ContextBegin, f.contextPrefixString(r.Context))
		fr.ContextEnd = fmt.Sprintf(fr.ContextEnd, f.contextSuffixString(r.Context))
	} else {
		*fr = FormatRecord{
			LogLevel:     f.levelFormatString(r.LogLevel),
			Time:         f.dateFormatString(r.Time),
			File:         f.fileFormatString(r.File),
			Line:         f.lineFormatString(r.Line),
			Message:      f.messageFormatString(r.Message),
			AppVersion:   f.versionFormatString(r.AppVersion),
			Context:      f.contextFormatStrings(r.Context),
			ContextBegin: f.contextPrefixString(r.Context),
			ContextEnd:   f.contextSuffixString(r.Context),
		}
	}

	return f.FinalString(r, fr), nil
}

type simpleFormatterBuilder struct {
	formatter simpleFormatter
}

var SimpleFormatter simpleFormatterBuilder = simpleFormatterBuilder{
	formatter: simpleFormatter{
		FormatterBase: FormatterBase{
			dateFormat: "",
			next:       nil,
		},
	},
}

func (b simpleFormatterBuilder) TimeFormat(format string) simpleFormatterBuilder {
	b.formatter.dateFormat = format
	return b
}
func (b simpleFormatterBuilder) Next(formatter Formatter) FormatterBuilder {
	b.formatter.next = formatter
	return b
}
func (b simpleFormatterBuilder) New() Formatter {
	if b.formatter.dateFormat == "" {
		b.formatter.dateFormat = "02/01/2006 15:04:05 -07:00"
	}

	t := &simpleFormatter{
		FormatterBase: b.formatter.FormatterBase,
	}

	return t
}

type consoleColorFormatter struct {
	FormatterBase
	colorScheme AnsiColorScheme
}

func (f *consoleColorFormatter) styleFor(nameID string) AnsiStyle {
	return f.colorScheme.GetStyle(nameID)
}

func (f *consoleColorFormatter) levelFormatString(l logLevel) string {
	return f.styleFor(l.CodeName()).Apply(" %s ")
}
func (f *consoleColorFormatter) dateFormatString(_ time.Time) string {
	return f.styleFor("time").Apply("%s")
}
func (f *consoleColorFormatter) fileFormatString(_ string) string {
	return f.styleFor("file").Apply("%s")
}
func (f *consoleColorFormatter) lineFormatString(_ int) string {
	return f.styleFor("line").Apply("%s")
}
func (f *consoleColorFormatter) messageFormatString(_ string) string {
	return f.styleFor("message").Apply("%s")
}
func (f *consoleColorFormatter) versionFormatString(_ version.Version) string {
	return f.styleFor("version").Apply("%s")
}
func (f *consoleColorFormatter) contextFormatStrings(ctx map[string]string) map[string]datatypes.SPair[string] {
	formatMap := make(map[string]datatypes.SPair[string], len(ctx))

	for k := range ctx {
		formatMap[k] = datatypes.NewSPair(
			f.styleFor("context-key").Apply("%s"),
			f.styleFor("context-value").Apply("%s"),
		)
	}
	return formatMap
}

func (f *consoleColorFormatter) Format(r *Record, formatRecord ...*FormatRecord) (string, error) {
	fr := &FormatRecord{}
	if len(formatRecord) > 0 {
		fr = formatRecord[0]
	}

	if f.Next() != nil {
		if _, err := f.Next().Format(r, fr); err != nil {
			return "", err
		}

		fr.LogLevel = fmt.Sprintf(fr.LogLevel, f.levelFormatString(r.LogLevel))
		fr.Time = fmt.Sprintf(fr.Time, f.dateFormatString(r.Time))
		fr.File = fmt.Sprintf(fr.File, f.fileFormatString(r.File))
		fr.Line = fmt.Sprintf(fr.Line, f.lineFormatString(r.Line))
		fr.Message = fmt.Sprintf(fr.Message, f.messageFormatString(r.Message))
		fr.AppVersion = fmt.Sprintf(fr.AppVersion, f.versionFormatString(r.AppVersion))
		fr.Context = mapx.Apply(fr.Context, func(k string, v datatypes.SPair[string]) datatypes.SPair[string] {
			last := f.contextFormatStrings(r.Context)[k]
			current := fr.Context[k]

			formatKey := fmt.Sprintf(current.First, last.First)
			formatValue := fmt.Sprintf(current.Second, last.Second)

			return datatypes.NewSPair(formatKey, formatValue)
		})
		fr.ContextBegin = fmt.Sprintf(fr.ContextBegin, f.contextPrefixString(r.Context))
		fr.ContextEnd = fmt.Sprintf(fr.ContextEnd, f.contextSuffixString(r.Context))
	} else {
		*fr = FormatRecord{
			LogLevel:     f.levelFormatString(r.LogLevel),
			Time:         f.dateFormatString(r.Time),
			File:         f.fileFormatString(r.File),
			Line:         f.lineFormatString(r.Line),
			Message:      f.messageFormatString(r.Message),
			AppVersion:   f.versionFormatString(r.AppVersion),
			Context:      f.contextFormatStrings(r.Context),
			ContextBegin: f.contextPrefixString(r.Context),
			ContextEnd:   f.contextSuffixString(r.Context),
		}
	}

	return f.FinalString(r, fr), nil
}

type ConsoleColorFormatterBuilder struct {
	formatter     consoleColorFormatter
	baseFormatter Formatter
}

var ConsoleColorFormatter ConsoleColorFormatterBuilder = ConsoleColorFormatterBuilder{
	formatter: consoleColorFormatter{
		colorScheme: Ansi.DefaultColorScheme(),
		FormatterBase: FormatterBase{
			dateFormat: "02/01/2006 15:04:05 -07:00",
			next:       nil,
		},
	},
	baseFormatter: SimpleFormatter.New(),
}

func (b ConsoleColorFormatterBuilder) BaseFormatter(formatter Formatter) ConsoleColorFormatterBuilder {
	b.baseFormatter = formatter
	return b
}
func (b ConsoleColorFormatterBuilder) AddColor(nameID string, colorCode AnsiStyle) ConsoleColorFormatterBuilder {
	if b.formatter.colorScheme.styles == nil {
		b.formatter.colorScheme.styles = make(map[string]AnsiStyle)
	}
	if _, exists := b.formatter.colorScheme.styles[nameID]; exists {
		// DEBUG
		fmt.Printf("Color with name '%s' is being overwritten\n", nameID)
	}

	b.formatter.colorScheme.styles[nameID] = colorCode
	return b
}
func (b ConsoleColorFormatterBuilder) DefaultColor(colorCode AnsiStyle) ConsoleColorFormatterBuilder {
	if b.formatter.colorScheme.styles == nil {
		b.formatter.colorScheme.styles = make(map[string]AnsiStyle)
	}

	b.formatter.colorScheme.styles["default"] = colorCode
	return b
}
func (b ConsoleColorFormatterBuilder) Next(formatter Formatter) FormatterBuilder {
	b.formatter.next = formatter
	return b
}
func (b ConsoleColorFormatterBuilder) New() Formatter {
	next := (Formatter)(nil)
	if b.formatter.next != nil {
		next = b.formatter.next
	}
	if b.baseFormatter == nil {
		b.baseFormatter = SimpleFormatter.New()
	}

	b.baseFormatter.SetNext(next)
	b.formatter.SetNext(b.baseFormatter)

	return &consoleColorFormatter{
		FormatterBase: b.formatter.FormatterBase,
		colorScheme:   b.formatter.colorScheme,
	}
}

var _ Formatter = (*simpleFormatter)(nil)
var _ Formatter = (*consoleColorFormatter)(nil)
var _ FormatterBuilder = (*simpleFormatterBuilder)(nil)
var _ FormatterBuilder = (*ConsoleColorFormatterBuilder)(nil)
