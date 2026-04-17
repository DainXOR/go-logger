package logger

import "fmt"

type AnsiCode string

func (c AnsiCode) String() string {
	return string(c)
}

const ( // Color constants
	TXT_BLACK   AnsiCode = "30m"
	TXT_RED     AnsiCode = "31m"
	TXT_GREEN   AnsiCode = "32m"
	TXT_YELLOW  AnsiCode = "33m"
	TXT_BLUE    AnsiCode = "34m"
	TXT_MAGENTA AnsiCode = "35m"
	TXT_CYAN    AnsiCode = "36m"
	TXT_WHITE   AnsiCode = "37m"

	BG_BLACK   AnsiCode = "40"
	BG_RED     AnsiCode = "41"
	BG_GREEN   AnsiCode = "42"
	BG_YELLOW  AnsiCode = "43"
	BG_BLUE    AnsiCode = "44"
	BG_MAGENTA AnsiCode = "45"
	BG_CYAN    AnsiCode = "46"
	BG_WHITE   AnsiCode = "47"

	CLR_START AnsiCode = "\033["
	CLR_RESET AnsiCode = "\033[0m"

	CLR_NONE AnsiCode = "" // No format
)

type ansiStyle struct{}

var Ansi = ansiStyle{}

func (ansiStyle) Debug() AnsiStyle {
	return AnsiStyle{Background: BG_GREEN, Text: TXT_BLACK}
}
func (ansiStyle) Info() AnsiStyle {
	return AnsiStyle{Background: BG_CYAN, Text: TXT_BLACK}
}
func (ansiStyle) Warn() AnsiStyle {
	return AnsiStyle{Background: BG_YELLOW, Text: TXT_BLACK}
}
func (ansiStyle) Error() AnsiStyle {
	return AnsiStyle{Background: BG_RED, Text: TXT_BLACK}
}
func (ansiStyle) Fatal() AnsiStyle {
	return AnsiStyle{Background: BG_RED, Text: TXT_WHITE}
}
func (ansiStyle) Deprecate() AnsiStyle {
	return AnsiStyle{Background: BG_WHITE, Text: TXT_MAGENTA}
}
func (ansiStyle) DeprecateWarning() AnsiStyle {
	return AnsiStyle{Background: BG_YELLOW, Text: TXT_MAGENTA}
}
func (ansiStyle) DeprecateError() AnsiStyle {
	return AnsiStyle{Background: BG_RED, Text: TXT_CYAN}
}
func (ansiStyle) DeprecateFatal() AnsiStyle {
	return AnsiStyle{Background: BG_RED, Text: TXT_WHITE}
}
func (ansiStyle) DeprecateReason() AnsiStyle {
	return AnsiStyle{Background: BG_YELLOW, Text: TXT_WHITE}
}
func (ansiStyle) Lava() AnsiStyle {
	return AnsiStyle{Background: BG_WHITE, Text: TXT_BLACK}
}
func (ansiStyle) ColdLava() AnsiStyle {
	return AnsiStyle{Background: BG_YELLOW, Text: TXT_BLACK}
}
func (ansiStyle) DriedLava() AnsiStyle {
	return AnsiStyle{Background: BG_RED, Text: TXT_BLACK}
}
func (ansiStyle) File() AnsiStyle {
	return AnsiStyle{Background: BG_BLUE, Text: TXT_WHITE, NoStop: true}
}
func (ansiStyle) Line() AnsiStyle {
	return AnsiStyle{Background: BG_BLUE, Text: TXT_WHITE}
}
func (ansiStyle) Default() AnsiStyle {
	return AnsiStyle{Background: CLR_NONE, Text: CLR_NONE}
}

type ColorScheme[S Style] interface {
	GetStyle(name string) S
}
type Style interface {
	Apply(text string) string
}

/*
AnsiColorScheme implements ColorScheme for ANSI styles
It provides a mapping of log levels to ANSI styles for console output.
It allows for easy customization of log output colors using identifiers.
The default identifiers used are listed here for reference:
  - debug
  - info
  - warning
  - error
  - fatal
  - deprecate
  - deprecate_warning
  - deprecate_error
  - deprecate_fatal
  - deprecate_reason
  - lava
  - lava_hot
  - lava_cold
  - lava_dry
  - time
  - file
  - line
  - version
  - message
  - context-key
  - context-value
  - default (used when no specific style is found)

You may add identifiers as needed for your custom formatters.
*/
type AnsiColorScheme struct {
	styles map[string]AnsiStyle
}

func (cs AnsiColorScheme) GetStyle(name string) AnsiStyle {
	if style, exists := cs.styles[name]; exists {
		return style
	}
	return AnsiStyle{Background: BG_BLACK, Text: TXT_WHITE} // Default style
}

type AnsiStyle struct {
	Background AnsiCode
	Text       AnsiCode
	NoStop     bool
}

func (s AnsiStyle) Apply(text string) string {
	if s.NoStop {
		return fmt.Sprintf("%s%s;%s%s", CLR_START, s.Background, s.Text, text)
	}

	return fmt.Sprintf("%s%s;%s%s%s", CLR_START, s.Background, s.Text, text, CLR_RESET)
}

var _ Style = (*AnsiStyle)(nil)
var _ ColorScheme[AnsiStyle] = (*AnsiColorScheme)(nil)

func (ansiStyle) DefaultColorScheme() AnsiColorScheme {
	return AnsiColorScheme{
		styles: map[string]AnsiStyle{
			"debug":   Ansi.Debug(),
			"info":    Ansi.Info(),
			"warning": Ansi.Warn(),
			"error":   Ansi.Error(),
			"fatal":   Ansi.Fatal(),

			"deprecate":         Ansi.Deprecate(),
			"deprecate_warning": Ansi.DeprecateWarning(),
			"deprecate_error":   Ansi.DeprecateError(),
			"deprecate_fatal":   Ansi.DeprecateFatal(),
			"deprecate_reason":  Ansi.DeprecateReason(),

			"lava":      Ansi.Lava(),
			"lava_hot":  Ansi.Lava(),
			"lava_cold": Ansi.ColdLava(),
			"lava_dry":  Ansi.DriedLava(),

			"file":    Ansi.File(),
			"line":    Ansi.Line(),
			"default": Ansi.Default(),
		},
	}
}
