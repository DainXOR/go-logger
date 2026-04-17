package logger

import (
	"fmt"

	"github.com/DainXOR/go-utils/cast"
	"github.com/DainXOR/go-utils/datatypes"
)

type logLevel struct {
	code     uint16
	name     string
	codeName string
}

func (l logLevel) Code() uint16 {
	return l.code
}

func (l logLevel) Name() string {
	return l.name
}
func (l logLevel) CodeName() string {
	return l.codeName
}

func createLogLevels() map[string]logLevel {
	LEVEL_DEBUG := logLevel{code: 1 << 0, name: "DEBUG", codeName: "debug"}
	LEVEL_INFO := logLevel{code: 1 << 1, name: "INFO", codeName: "info"}
	LEVEL_WARNING := logLevel{code: 1 << 2, name: "WARNING", codeName: "warning"}
	LEVEL_ERROR := logLevel{code: 1 << 3, name: "ERROR", codeName: "error"}
	LEVEL_FATAL := logLevel{code: 1 << 4, name: "FATAL", codeName: "fatal"}

	LEVEL_DEPRECATE := logLevel{code: 1 << 8, name: "DEPRECATE", codeName: "deprecate"}
	LEVEL_DEPRECATE_WARNING := logLevel{code: LEVEL_DEPRECATE.code | LEVEL_WARNING.code, name: LEVEL_WARNING.Name() + ":DEPRECATE", codeName: "deprecate_warning"}
	LEVEL_DEPRECATE_ERROR := logLevel{code: LEVEL_DEPRECATE.code | LEVEL_ERROR.code, name: LEVEL_ERROR.Name() + ":DEPRECATE", codeName: "deprecate_error"}
	LEVEL_DEPRECATE_FATAL := logLevel{code: LEVEL_DEPRECATE.code | LEVEL_FATAL.code, name: LEVEL_FATAL.Name() + ":DEPRECATE", codeName: "deprecate_fatal"}

	LEVEL_LAVA := logLevel{code: 1 << 9, name: "LAVA", codeName: "lava"}
	LEVEL_LAVA_HOT := logLevel{code: LEVEL_LAVA.code | LEVEL_DEBUG.code, name: LEVEL_LAVA.Name() + ":HOT", codeName: "lava_hot"}
	LEVEL_LAVA_COLD := logLevel{code: LEVEL_LAVA.code | LEVEL_WARNING.code, name: LEVEL_LAVA.Name() + ":COLD", codeName: "lava_cold"}
	LEVEL_LAVA_DRY := logLevel{code: LEVEL_LAVA.code | LEVEL_ERROR.code, name: LEVEL_LAVA.Name() + ":DRY", codeName: "lava_dry"}

	LEVEL_ALL := logLevel{code: ^uint16(0), name: "ALL", codeName: "all"}
	LEVEL_NONE := logLevel{code: 0, name: "NONE", codeName: "none"}

	LEVEL_URGENCY := logLevel{code: LEVEL_DEBUG.code | LEVEL_INFO.code | LEVEL_WARNING.code | LEVEL_ERROR.code | LEVEL_FATAL.code, name: "URGENCY", codeName: "urgency"}
	LEVEL_TYPES := logLevel{code: LEVEL_DEPRECATE.code | LEVEL_LAVA.code, name: "TYPES", codeName: "types"}

	return map[string]logLevel{
		LEVEL_DEBUG.codeName:   LEVEL_DEBUG,
		LEVEL_INFO.codeName:    LEVEL_INFO,
		LEVEL_WARNING.codeName: LEVEL_WARNING,
		LEVEL_ERROR.codeName:   LEVEL_ERROR,
		LEVEL_FATAL.codeName:   LEVEL_FATAL,

		LEVEL_DEPRECATE.codeName:         LEVEL_DEPRECATE,
		LEVEL_DEPRECATE_WARNING.codeName: LEVEL_DEPRECATE_WARNING,
		LEVEL_DEPRECATE_ERROR.codeName:   LEVEL_DEPRECATE_ERROR,
		LEVEL_DEPRECATE_FATAL.codeName:   LEVEL_DEPRECATE_FATAL,

		LEVEL_LAVA.codeName:      LEVEL_LAVA,
		LEVEL_LAVA_HOT.codeName:  LEVEL_LAVA_HOT,
		LEVEL_LAVA_COLD.codeName: LEVEL_LAVA_COLD,
		LEVEL_LAVA_DRY.codeName:  LEVEL_LAVA_DRY,

		LEVEL_ALL.codeName:  LEVEL_ALL,
		LEVEL_NONE.codeName: LEVEL_NONE,

		LEVEL_URGENCY.codeName: LEVEL_URGENCY,
		LEVEL_TYPES.codeName:   LEVEL_TYPES,
	}
}

var logLevels = createLogLevels()

func (l logLevel) Has(level logLevel) bool {
	return l.code&level.code == level.code
}
func (l logLevel) Is(level logLevel) bool {
	return l.code == level.code
}
func (l logLevel) IsLowerThan(level logLevel) bool {
	return l.code < level.code
}
func (l logLevel) IsHigherThan(level logLevel) bool {
	return l.code > level.code
}

func (l logLevel) And(level logLevel) logLevel {
	l1 := Level.Highest(l, level)

	return logLevel{code: l.code | level.code, name: l1.name, codeName: l1.codeName}
}
func (l logLevel) Not(level logLevel) logLevel {
	newlevel := logLevel{code: l.code &^ level.code, name: l.name, codeName: l.codeName}
	dominant := Level.Dominant(newlevel)
	newlevel.name = dominant.name
	newlevel.codeName = dominant.codeName

	return newlevel
}

func (l *logLevel) Set(level logLevel) *logLevel {
	*l = l.And(level)

	return l
}
func (l *logLevel) Unset(level logLevel) *logLevel {
	*l = l.Not(level)

	return l
}

func (l *logLevel) AsMax() logLevel {
	dom := Level.Dominant(*l)
	mask := dom.code

	for i := l.code; i > 0; i <<= 1 {
		mask |= i
	}

	return logLevel{code: mask, name: dom.name, codeName: dom.codeName}
}
func (l *logLevel) AsMin() logLevel {
	dom := Level.Dominant(*l)
	mask := dom.code

	for i := l.code; i > 0; i >>= 1 {
		mask |= i
	}

	return logLevel{code: mask, name: dom.name, codeName: dom.codeName}
}

func (l *logLevel) Select(level logLevel) logLevel {
	if l.Has(level) {
		return level
	}
	return Level.None()
}

func (l logLevel) UrgencyLevels() uint16 {
	return logLevels["urgency"].code & l.code
}
func (l logLevel) TypeLevels() uint16 {
	return uint16(cast.BoolToInt(l.Has(Level.Lava())))*Level.Lava().code | uint16(cast.BoolToInt(l.Has(Level.Deprecate())))*Level.Deprecate().code
}
func (l logLevel) IsValid() bool {
	return l.Is(Level.None()) || Level.All().Has(l)
}

type iLevel struct{}

var Level iLevel

func (iLevel) Get(nameID string) (logLevel, error) {
	if level, ok := logLevels[nameID]; ok {
		return level, nil
	}

	return Level.None(), fmt.Errorf("Invalid log level: %s", nameID)
}
func (iLevel) ContainedIn(l logLevel) []logLevel {
	var contained []logLevel
	for _, level := range logLevels {
		if l.Has(level) {
			contained = append(contained, level)
		}
	}
	return contained
}
func (iLevel) Combine(levels ...logLevel) logLevel {
	if len(levels) == 0 {
		return Level.None()
	}

	combined := Level.None()
	for _, level := range levels {
		combined = combined.And(level)
	}
	return combined
}

func (iLevel) Debug() logLevel {
	return logLevels["debug"]
}
func (iLevel) Info() logLevel {
	return logLevels["info"]
}
func (iLevel) Warning() logLevel {
	return logLevels["warning"]
}
func (iLevel) Error() logLevel {
	return logLevels["error"]
}
func (iLevel) Fatal() logLevel {
	return logLevels["fatal"]
}

func (iLevel) Deprecate() logLevel {
	return logLevels["deprecate"]
}
func (iLevel) DeprecateWarning() logLevel {
	return logLevels["deprecate_warning"]
}
func (iLevel) DeprecateError() logLevel {
	return logLevels["deprecate_error"]
}
func (iLevel) DeprecateFatal() logLevel {
	return logLevels["deprecate_fatal"]
}

func (iLevel) Lava() logLevel {
	return logLevels["lava"]
}
func (iLevel) LavaHot() logLevel {
	return logLevels["lava_hot"]
}
func (iLevel) LavaCold() logLevel {
	return logLevels["lava_cold"]
}
func (iLevel) LavaDry() logLevel {
	return logLevels["lava_dry"]
}

func (iLevel) All() logLevel {
	return logLevels["all"]
}
func (iLevel) None() logLevel {
	return logLevels["none"]
}

// Returns the higher of two log levels based on their code values.
// If both levels are equal, it returns the first one.
func (iLevel) Highest(levels ...logLevel) logLevel {
	if len(levels) == 0 {
		return Level.None()
	}

	highest := Level.None()
	for _, level := range levels {
		if level.code > highest.code {
			highest = level
		}
	}
	return highest
}
func (iLevel) Lowest(levels ...logLevel) logLevel {
	if len(levels) == 0 {
		return Level.None()
	}

	lowest := Level.All()
	for _, level := range levels {
		if level.code < lowest.code {
			lowest = level
		}
	}
	return lowest
}
func (iLevel) InOrder(l1, l2 logLevel) (logLevel, logLevel) {
	if l1.code < l2.code {
		return l1, l2
	}
	return l2, l1
}
func (iLevel) Dominant(l logLevel) logLevel {
	if l.Has(Level.Fatal()) {
		return Level.Fatal()
	}
	if l.Has(Level.Error()) {
		return Level.Error()
	}
	if l.Has(Level.Warning()) {
		return Level.Warning()
	}
	if l.Has(Level.Info()) {
		return Level.Info()
	}
	if l.Has(Level.Debug()) {
		return Level.Debug()
	}
	return Level.None()
}

func (iLevel) FromCode(code uint16) (logLevel, error) {
	for _, level := range logLevels {
		if level.code == code {
			return level, nil
		}
	}
	return Level.None(), fmt.Errorf("Invalid log level code: %d", code)
}
func (iLevel) InterpretCode(code uint16) logLevel {
	combined := Level.None()

	for _, level := range logLevels {
		if code&level.code == level.code {
			combined = combined.And(level)
		}
	}
	return combined
}

type logFlag uint16

func logFlags() map[string]logFlag {
	const (
		FLAG_DATE logFlag = 1 << iota
		FLAG_TIME
		FLAG_FILE
		FLAG_LINE
		FLAG_APP_VERSION
		FLAG_COLOR
		FLAG_CONTEXT

		FLAG_ALL  logFlag = ^logFlag(0)
		FLAG_NONE logFlag = 0
	)

	return map[string]logFlag{
		"date":        FLAG_DATE,
		"time":        FLAG_TIME,
		"file":        FLAG_FILE,
		"line":        FLAG_LINE,
		"app_version": FLAG_APP_VERSION,
		"color":       FLAG_COLOR,
		"context":     FLAG_CONTEXT,

		"all":  FLAG_ALL,
		"none": FLAG_NONE,
	} // Default to no flags
}

type iFlag struct{}

var Flag iFlag

func (iFlag) DateTime() logFlag {
	return logFlags()["date"]
}
func (iFlag) Time() logFlag {
	return logFlags()["time"]
}
func (iFlag) File() logFlag {
	return logFlags()["file"]
}
func (iFlag) Line() logFlag {
	return logFlags()["line"]
}
func (iFlag) AppVersion() logFlag {
	return logFlags()["app_version"]
}
func (iFlag) Color() logFlag {
	return logFlags()["color"]
}
func (iFlag) Context() logFlag {
	return logFlags()["context"]
}
func (iFlag) All() logFlag {
	return logFlags()["all"]
}
func (iFlag) None() logFlag {
	return logFlags()["none"]
}

func (f logFlag) Has(flag logFlag) bool {
	return (f & flag) != 0
}
func (f logFlag) Set(flag logFlag) logFlag {
	return f | flag
}
func (f logFlag) Unset(flag logFlag) logFlag {
	return f &^ flag
}

type logInternalVal string
type iInternal struct{}

var internal iInternal

func (iInternal) prefix(txt string) string {
	return "_internal_" + txt
}
func (iInternal) CallOriginOffset() logInternalVal {
	return logInternalVal(internal.prefix("call_origin_offset"))
}
func (iInternal) FormatString() logInternalVal {
	return logInternalVal(internal.prefix("format_string"))
}
func (iInternal) AppVersion() logInternalVal {
	return logInternalVal(internal.prefix("app_version"))
}

func (l logInternalVal) String() string {
	return string(l)
}
func (l logInternalVal) Value(val string) datatypes.SPair[string] {
	return datatypes.NewSPair(l.String(), val)
}
func (l logInternalVal) Check(val string) bool {
	return (len(val) > 10) && (val[:10] == "_internal_") && (logInternalVal(val) == l)
}

type iTerminationCode struct{}

var TerminationCode iTerminationCode

func (iTerminationCode) Normal() int {
	return 0
}
func (iTerminationCode) WriteError() int {
	return 0x0106_FA11
}
func (iTerminationCode) FormatError() int {
	return 0x0FA7_FA11
}
func (iTerminationCode) MaxAttemptsExceeded() int {
	return 0xECCD_A775
}
func (iTerminationCode) Deprecated() int {
	return 0xDEAD_C0DE
}
