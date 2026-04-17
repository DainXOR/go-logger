package logger

import (
	"errors"
	"maps"
	"os"
	"strconv"
	"strings"

	"github.com/DainXOR/go-utils/collections"
	"github.com/DainXOR/go-utils/datatypes"
	"github.com/DainXOR/go-utils/reflex"
	"github.com/DainXOR/go-utils/stringx"
	"github.com/DainXOR/go-utils/version"

	"fmt"

	_ "github.com/joho/godotenv/autoload"
)

type dnxLogger struct {
	configurations

	logAttempts   uint8 // Number of log attempts, used to prevent infinite loops in logging
	usingDefaults bool  // Indicates if the logger is using default settings

	abnormalWriters map[string]outputBinding // Special writers used when normal logging is failing
}

var dnxGlobalLogger *dnxLogger

func init() {
	dnxGlobalLogger = NewWithEnv()
}

// Returns the singleton instance of dnxLogger, initializing it if necessary.
// This function should be used to access the global logger throughout the instanceless methods.
// It ensures that the logger is initialized before use.
func get() *dnxLogger {
	if dnxGlobalLogger == nil {
		dnxGlobalLogger = NewDefault()
	}
	return dnxGlobalLogger
}
func clone(original *dnxLogger) *dnxLogger {
	logger := &dnxLogger{
		configurations:  original.configurations,
		logAttempts:     0,
		usingDefaults:   false,
		abnormalWriters: make(map[string]outputBinding, len(original.abnormalWriters)),
	}

	maps.Copy(logger.abnormalWriters, original.abnormalWriters)
	logger.Debug("Logger cloned")
	return logger
}

func NewClone() *dnxLogger {
	return clone(get())
}
func NewDefault() *dnxLogger {
	logger := &dnxLogger{
		configurations: *NewConfigs(),
		logAttempts:    0,
		usingDefaults:  true,

		abnormalWriters: map[string]outputBinding{
			"console": {writer: ConsoleWriter.NewLine().New(), formatter: SimpleFormatter.New()},
		},
	}

	logger.Debug("Logger created")

	return logger
}
func NewWithEnv() *dnxLogger {
	logger := NewDefault()
	LoadEnv(logger)
	return logger
}
func LoadEnv(logger *dnxLogger) {
	logger.Debug("Loading environment variables for logger")

	minLogLevel, existMinLevel := os.LookupEnv("DNX_LOG_MIN_LEVEL")
	disableLevels, existDisableLevels := os.LookupEnv("DNX_LOG_DISABLE_LEVELS")
	logConsole, existLogConsole := os.LookupEnv("DNX_LOG_CONSOLE")
	logFile, existLogFile := os.LookupEnv("DNX_LOG_FILE")
	logFilePath, existLogFilePath := os.LookupEnv("DNX_LOG_FILE_PATH")
	logWithColor, existLogWithColor := os.LookupEnv("DNX_LOG_WITH_COLOR")

	if existMinLevel {
		l, err := Level.Get(strings.ToLower(minLogLevel))
		if err == nil {
			logger.SetMinLogLevel(l)
		}
	}
	if existDisableLevels {
		levelsIDs := strings.Split(strings.ToLower(disableLevels), "|")
		lvls, _ := collections.MapE(levelsIDs, Level.Get, true)

		logger.DisableLogLevels(Level.Combine(lvls...))
	}
	if existLogConsole && logConsole == "FALSE" {
		logger.RemoveWriter("console")
	}
	if existLogFile && logFile == "TRUE" {
		if !existLogFilePath {
			logFilePath = ""
		}
		logger.AddWriter("file", FileWriter.FilePath(logFilePath).NewLine().New(), SimpleFormatter.New())
	}
	if existLogWithColor && logWithColor == "TRUE" {
		logger.ChangeFormatter("console", ConsoleColorFormatter.New())
	}

	logger.Debug("Logger environment variables loaded")
}

func (i *dnxLogger) Close() {
	i.Debug("Closing writers")
	i.Lava(version.V("0.1.5"), "writer.Close() error return is not handled")
	for _, writer := range i.writers {
		writer.writer.Close()
	}
	for _, writer := range i.abnormalWriters {
		writer.writer.Close()
	}
}
func Close() {
	get().Close()
}

func panicOrError(msg string, condition bool) error {
	if condition {
		panic(msg)
	} else {
		return errors.New(msg)
	}
}
func (i *dnxLogger) logLevelsNames() string {
	urgencyLevels := Level.InterpretCode(i.LogLevels().UrgencyLevels())
	contained := Level.ContainedIn(urgencyLevels)
	names := collections.Map(contained, logLevel.Name)

	return "| " + strings.Join(names, " | ") + " |"
}

func (i *dnxLogger) registerLogAttempt() bool {
	i.logAttempts++

	if i.logAttempts > i.warningLogAttempts && i.logAttempts < i.maxLogAttempts {
		i.internalAbnormalWrite(NewRecord("Too many log attempts, will panic if this continues"))
		return false

	} else if i.logAttempts >= i.maxLogAttempts {
		i.internalAbnormalWrite(NewRecord("Terminating program due to too many log attempts"))
		if i.panicOnMaxAttempts {
			panic(TerminationCode.MaxAttemptsExceeded())
		}

		return false
	}
	return true
}
func (i *dnxLogger) resetLogAttempts() bool {
	if i.logAttempts == 1 {
		i.logAttempts = 0
		return true

	} else if i.logAttempts > 1 {
		i.logAttempts = 0
		i.internalAbnormalWrite(NewRecord("Resetting log attempts after too many attempts"))
		return true
	}

	return false
}

func (i *dnxLogger) SetVersion(version version.Version) {
	i.appVersion = version
	i.Debug("App version changed to: ", version.String(),
		internal.CallOriginOffset().Value("1"),
	)
}
func SetVersion(version version.Version) {
	get().SetVersion(version)
}

// Returns the application version used as reference for deprecation warnings.
func (i *dnxLogger) AppVersion() version.Version {
	return i.appVersion
}
func AppVersion() version.Version {
	return get().AppVersion()
}

// Returns the current logging levels loaded as bitmask
func (i *dnxLogger) LogLevels() logLevel {
	return i.logLevels
}
func LogLevels() logLevel {
	return get().LogLevels()
}

func (i *dnxLogger) SetLogLevels(options logLevel) bool {
	if !options.IsValid() {
		i.Warning("Invalid logging options")
		return false
	} else if options.Is(Level.All()) {
		i.Debug("Logging options set to:", Level.All().Name())
		i.logLevels.Set(Level.All())
		return true
	} else if options.Is(Level.None()) {
		i.Debug("Logging options set to:", Level.None().Name())
		i.logLevels.Set(Level.None())
		return true
	}

	i.logLevels.Set(options)
	i.Debug("Logging options set to:", i.logLevelsNames())
	return true
}
func (i *dnxLogger) EnableLogLevels(options logLevel) bool {
	if !options.IsValid() {
		i.Warning("Invalid logging option")
		return false
	}

	i.SetLogLevels(i.LogLevels().And(options))
	return true
}
func (i *dnxLogger) DisableLogLevels(options logLevel) bool {
	if !options.IsValid() {
		i.Warning("Invalid logging option")
		return false
	}

	i.SetLogLevels(i.LogLevels().Not(options))
	return true
}
func (i *dnxLogger) SetMinLogLevel(level logLevel) bool {
	if !level.IsValid() {
		i.Warning("Invalid logging level")
		return false
	}

	tmp := level.Select(Level.Deprecate()).And(level.Select(Level.Lava()))
	i.SetLogLevels(level.AsMin().And(tmp))

	return true
}
func SetLogLevels(options logLevel) bool {
	return get().SetLogLevels(options)
}
func EnableLogLevels(options logLevel) bool {
	return get().EnableLogLevels(options)
}
func DisableLogLevels(options logLevel) bool {
	return get().DisableLogLevels(options)
}
func SetMinLogLevel(level logLevel) bool {
	return get().SetMinLogLevel(level)
}

func (i *dnxLogger) EnableDeprecate() bool {
	return i.EnableLogLevels(Level.Deprecate())
}
func (i *dnxLogger) DisableDeprecate() bool {
	return i.DisableLogLevels(Level.Deprecate())
}
func (i *dnxLogger) EnableLava() bool {
	return i.EnableLogLevels(Level.Lava())
}
func (i *dnxLogger) DisableLava() bool {
	return i.DisableLogLevels(Level.Lava())
}
func EnableDeprecate() bool {
	return get().EnableDeprecate()
}
func DisableDeprecate() bool {
	return get().DisableDeprecate()
}
func EnableLava() bool {
	return get().EnableLava()
}
func DisableLava() bool {
	return get().DisableLava()
}

func (i *dnxLogger) canLog(level logLevel) bool {
	return i.LogLevels().Has(level)
}

func (i *dnxLogger) internalWrite(record Record) {
	if !i.canLog(record.LogLevel) {
		return
	}

	i.registerLogAttempt()
	removeList := make([]string, 0, len(i.writers))

	for p, pair := range i.writers {
		w, f := pair.writer, pair.formatter

		if w == nil || f == nil {
			record := i.generateRecord(Level.Error(), "Invalid writer or formatter in writer list, marking for removal")
			i.internalAbnormalWrite(record)
			removeList = append(removeList, p)
			continue
		}

		if str, err := f.Format(&record); err == nil {
			if err = w.Write(str); err != nil {
				record := i.generateRecord(Level.Error(), "Error during write:", err.Error())
				i.internalAbnormalWrite(record)
			}

		} else {
			record := i.generateRecord(Level.Error(), "Error formatting record:", err.Error())
			i.internalAbnormalWrite(record)
		}
	}

	i.RemoveWriters(removeList...)
	i.resetLogAttempts()
}
func (i *dnxLogger) internalAbnormalWrite(record Record) error {
	if !i.canLog(record.LogLevel) {
		return nil
	}

	for _, wf := range get().abnormalWriters {
		if wf.writer == nil || wf.formatter == nil {
			return panicOrError("Invalid writer or formatter in abnormal write", i.canPanicOnAbnormalWrite)
		}

		w, f := wf.writer, wf.formatter
		if str, err := f.Format(&record); err == nil {
			if err = w.Write(str); err != nil {
				return panicOrError("Error during abnormal write: "+err.Error(), i.canPanicOnAbnormalWrite)
			}

		} else {
			return panicOrError("Error formatting record for abnormal write: "+err.Error(), i.canPanicOnAbnormalWrite)
		}
	}

	return nil
}

func (i *dnxLogger) generateRecord(level logLevel, v ...any) Record {
	msg := ""
	extraParts := []datatypes.SPair[string]{
		internal.CallOriginOffset().Value("4"),
		internal.AppVersion().Value(i.AppVersion().String()),
	}

	for _, pair := range reflex.ValuesOfType[datatypes.SPair[string]](v) {
		if internal.CallOriginOffset().Check(pair.First) {
			offset, _ := strconv.Atoi(pair.Second)
			originalOffset, _ := strconv.Atoi(extraParts[0].Second)
			extraParts[0].Second = fmt.Sprint(originalOffset + offset)
			continue
		} else if internal.FormatString().Check(pair.First) {
			msg = fmt.Sprintf(pair.Second, reflex.ExcludeOfType[datatypes.SPair[string]](v)...)
			continue
		} else if internal.AppVersion().Check(pair.First) {
			version, err := version.VersionFrom(pair.Second)
			if err == nil {
				extraParts[1].Second = version.String()
			}
			continue
		}

		extraParts = append(extraParts, pair)
	}

	if msg == "" {
		msg = fmt.Sprintln(reflex.ExcludeOfType[datatypes.SPair[string]](v)...)
		msg = strings.TrimSuffix(msg, "\n")
	}

	record := NewRecord(msg, extraParts...)
	record.LogLevel = level
	return record
}

func (i *dnxLogger) iWrite(l logLevel, v ...any) {
	record := i.generateRecord(l, v...)
	i.internalWrite(record)
}
func (i *dnxLogger) iWritef(l logLevel, format string, v ...any) {
	extraParts := []any{datatypes.NewSPair(internal.FormatString().String(), format)}
	extraParts = append(extraParts, v...)

	record := i.generateRecord(l, extraParts...)
	i.internalWrite(record)
}
func (i *dnxLogger) iWriter(r Record) {
	if r.AppVersion == version.V0() {
		r.AppVersion = i.AppVersion()
	}
	i.internalWrite(r)
}

func (i *dnxLogger) Debug(v ...any) {
	i.iWrite(Level.Debug(), v...)
}
func (i *dnxLogger) Debugf(format string, v ...any) {
	i.iWritef(Level.Debug(), format, v...)
}
func (i *dnxLogger) Debugr(r Record) {
	r.LogLevel = Level.Debug()
	i.iWriter(r)
}
func Debug(v ...any) {
	get().Debug(v...)
}
func Debugf(format string, v ...any) {
	get().Debugf(format, v...)
}
func Debugr(r Record) {
	get().Debugr(r)
}

func (i *dnxLogger) Info(v ...any) {
	i.iWrite(Level.Info(), v...)
}
func (i *dnxLogger) Infof(format string, v ...any) {
	i.iWritef(Level.Info(), format, v...)
}
func (i *dnxLogger) Infor(r Record) {
	r.LogLevel = Level.Info()
	i.iWriter(r)
}
func Info(v ...any) {
	get().Info(v...)
}
func Infof(format string, v ...any) {
	get().Infof(format, v...)
}
func Infor(r Record) {
	get().Infor(r)
}

func (i *dnxLogger) Warning(v ...any) {
	i.iWrite(Level.Warning(), v...)
}
func (i *dnxLogger) Warningf(format string, v ...any) {
	i.iWritef(Level.Warning(), format, v...)
}
func (i *dnxLogger) Warningr(r Record) {
	r.LogLevel = Level.Warning()
	i.iWriter(r)
}
func Warning(v ...any) {
	get().Warning(v...)
}
func Warningf(format string, v ...any) {
	get().Warningf(format, v...)
}
func Warningr(r Record) {
	get().Warningr(r)
}

func (i *dnxLogger) Error(v ...any) {
	i.iWrite(Level.Error(), v...)
}
func (i *dnxLogger) Errorf(format string, v ...any) {
	i.iWritef(Level.Error(), format, v...)
}
func (i *dnxLogger) Errorr(r Record) {
	r.LogLevel = Level.Error()
	i.iWriter(r)
}
func Error(v ...any) {
	get().Error(v...)
}
func Errorf(format string, v ...any) {
	get().Errorf(format, v...)
}
func Errorr(r Record) {
	get().Errorr(r)
}

func (i *dnxLogger) Fatal(v ...any) {
	i.iWrite(Level.Fatal(), v...)
}
func (i *dnxLogger) Fatalf(format string, v ...any) {
	i.iWritef(Level.Fatal(), format, v...)
}
func (i *dnxLogger) Fatalr(r Record) {
	r.LogLevel = Level.Fatal()
	i.iWriter(r)
}
func Fatal(v ...any) {
	get().Fatal(v...)
}
func Fatalf(format string, v ...any) {
	get().Fatalf(format, v...)
}
func Fatalr(r Record) {
	get().Fatalr(r)
}

func (i *dnxLogger) Deprecate(deprecatedVersion version.Version, removalVersion version.Version, v ...any) (bool, error) {
	args := stringx.Join(v, " ")

	if i.AppVersion().GreaterEq(removalVersion.Plus(version.V("0.1.0"))) {
		i.iWrite(Level.DeprecateFatal(), "This feature has been removed in version:", removalVersion)
		i.iWrite(Level.DeprecateFatal(), "Reason:", args)
		i.iWrite(Level.DeprecateFatal(), "Unrecoverable deprecation error, application will terminate")
		panic(TerminationCode.Deprecated())

	} else if i.AppVersion().GreaterEq(removalVersion) {
		i.iWrite(Level.DeprecateError(), "This feature has been removed in version:", removalVersion)
		i.iWrite(Level.DeprecateError(), "Reason:", args)
		return false, fmt.Errorf("feature removed in version %s. %s", removalVersion, args)

	} else if i.AppVersion().GreaterEq(deprecatedVersion) && i.AppVersion().LessThan(removalVersion) {
		i.iWrite(Level.DeprecateWarning(), "This feature is marked for removal in version:", removalVersion)
		i.iWrite(Level.DeprecateWarning(), "Reason:", args)
		return false, fmt.Errorf("feature deprecated in version %s, will be removed in version %s. %s", deprecatedVersion, removalVersion, args)

	} else if i.AppVersion().GreaterEq(deprecatedVersion) {
		i.iWrite(Level.Deprecate(), "This feature will be removed in future versions")
		i.iWrite(Level.Deprecate(), "Reason:", args)
		return true, fmt.Errorf("feature deprecated in version %s. %s", deprecatedVersion, args)
	}
	return true, nil
}
func (i *dnxLogger) DeprecateMsg(deprecatedVersion version.Version, removalVersion version.Version, v ...any) string {
	_, err := i.Deprecate(deprecatedVersion, removalVersion, v...)

	if err != nil {
		return err.Error()
	}
	return ""
}
func Deprecate(deprecatedVersion version.Version, removalVersion version.Version, v ...any) (bool, error) {
	return get().Deprecate(deprecatedVersion, removalVersion, v...)
}
func DeprecateMsg(deprecatedVersion version.Version, removalVersion version.Version, v ...any) string {
	return get().DeprecateMsg(deprecatedVersion, removalVersion, v...)
}

type volcano struct {
	logger         *dnxLogger
	initialVersion version.Version
	coldVersion    version.Version
	driedVersion   version.Version
}

func (i *dnxLogger) Lava(v version.Version, values ...any) volcano {
	args := stringx.Join(values, " ")

	coldVersion := v.Plus(version.V("0.1.0"))
	driedVersion := v.Plus(version.V("0.2.0"))

	if i.AppVersion().GreaterEq(driedVersion) {
		i.iWrite(Level.LavaDry(), "Running code that should be removed, cleaned up or refactored.")
		i.iWrite(Level.LavaDry(), "Reason:", args)
		//iLogWarning(false, 1, lavaTxt, "Running code that should be removed, cleaned up or refactored.", args)
	} else if i.AppVersion().GreaterEq(coldVersion) {
		i.iWrite(Level.LavaCold(), "This code must be refactored.")
		i.iWrite(Level.LavaCold(), "Reason:", args)
		//iLogWarning(false, 1, coldLavaTxt, "This code must be refactored.", args)
	} else if i.AppVersion().GreaterEq(v) {
		i.iWrite(Level.LavaHot(), "This code should not be running as is, it is likely a bug.")
		i.iWrite(Level.LavaHot(), "Reason:", args)
		//iLogError(false, 1, driedLavaTxt, "This code should not be running as is, it is likely a bug.", args)
	}

	return volcano{
		logger:         i,
		initialVersion: v,
		coldVersion:    coldVersion,
		driedVersion:   driedVersion,
	}
}
func Lava(version version.Version, v ...any) volcano {
	return get().Lava(version, v...)
}
func (v *volcano) LavaStart() {
	if v.logger.AppVersion().GreaterEq(v.driedVersion) {
		v.logger.iWrite(Level.LavaDry(), "Start of lava cast")
	} else if v.logger.AppVersion().GreaterEq(v.coldVersion) {
		v.logger.iWrite(Level.LavaCold(), "Start of flow")
	} else if v.logger.AppVersion().GreaterEq(v.initialVersion) {
		v.logger.iWrite(Level.LavaHot(), "Start of flow")
	}
}
func (v *volcano) LavaEnd() {
	if v.logger.AppVersion().GreaterEq(v.driedVersion) {
		v.logger.iWrite(Level.LavaDry(), "End of lava cast")
	} else if v.logger.AppVersion().GreaterEq(v.coldVersion) {
		v.logger.iWrite(Level.LavaCold(), "End of flow")
	} else if v.logger.AppVersion().GreaterEq(v.initialVersion) {
		v.logger.iWrite(Level.LavaHot(), "End of flow")
	}
}
