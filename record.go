package logger

import (
	"strconv"
	"time"

	"github.com/DainXOR/go-utils/datatypes"
	"github.com/DainXOR/go-utils/runtimex"
	"github.com/DainXOR/go-utils/version"
)

type Record struct {
	LogLevel   logLevel
	Time       time.Time
	Message    string
	File       string
	Line       int
	AppVersion version.Version
	Context    map[string]string
}

func NewRecord(msg string, extra ...datatypes.SPair[string]) Record {
	rec := Record{
		Time:    time.Now(),
		Message: msg,
	}

	_, rec.File, rec.Line = runtimex.CallerInfo(2)
	rec.AppVersion = version.V0()
	rec.Context = make(map[string]string, len(extra))

	if len(extra) > 0 {
		for _, pair := range extra {
			if internal.AppVersion().Check(pair.First) {
				// If the key is app_version, convert the value to a Version type
				if version, err := version.VersionFrom(pair.Second); err == nil {
					rec.AppVersion = version
				}
				continue
			}
			if internal.CallOriginOffset().Check(pair.First) {
				if i, err := strconv.Atoi(pair.Second); err == nil {
					_, rec.File, rec.Line = runtimex.CallerInfo(2 + int(i))
				} else {
					rec.File, rec.Line = "UnknownFile", 0
				}
				continue
			}

			rec.Context[pair.First] = pair.Second
		}
	}

	return rec
}

type FormatRecord struct {
	LogLevel     string
	Time         string
	File         string
	Line         string
	Message      string
	AppVersion   string
	Context      map[string](datatypes.SPair[string])
	ContextBegin string
	ContextEnd   string
}
