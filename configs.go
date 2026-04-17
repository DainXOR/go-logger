package logger

import "github.com/DainXOR/go-utils/version"

type outputBinding struct {
	writer    Writer
	formatter Formatter
}

type configurations struct {
	logLevels logLevel
	logFlags  logFlag

	maxLogAttempts     uint8
	warningLogAttempts uint8

	panicOnMaxAttempts      bool
	canPanicOnAbnormalWrite bool

	appVersion version.Version

	writers map[string]outputBinding
}

// NewConfigs initializes a new configs instance with default values
func NewConfigs() *configurations {
	return &configurations{
		logLevels: Level.All(),
		logFlags:  Flag.DateTime() | Flag.File() | Flag.Line() | Flag.AppVersion(),

		warningLogAttempts: 10,
		maxLogAttempts:     15,

		panicOnMaxAttempts:      true,
		canPanicOnAbnormalWrite: true,

		appVersion: version.V("0.1.0"),

		writers: map[string]outputBinding{
			"console": {writer: ConsoleWriter.NewLine().New(), formatter: SimpleFormatter.New()},
		},
	}
}

func (c *configurations) NoWriters() *configurations {
	c.writers = make(map[string]outputBinding)
	return c
}
func (c *configurations) AddWriter(nameID string, writer Writer, formatter Formatter) *configurations {
	c.writers[nameID] = outputBinding{writer: writer, formatter: formatter}
	return c
}
func (c *configurations) RemoveWriter(nameID string) *configurations {
	if _, exists := c.writers[nameID]; !exists {
		return c
	}

	c.writers[nameID].writer.Close()
	delete(c.writers, nameID)
	return c
}
func (c *configurations) RemoveWriters(nameIDs ...string) *configurations {
	if len(nameIDs) == 0 {
		return c
	}

	for _, nameID := range nameIDs {
		if binding, exists := c.writers[nameID]; exists {
			binding.writer.Close()
			delete(c.writers, nameID)
		}
	}
	return c
}

func (c *configurations) ChangeWriter(nameID string, writer Writer) *configurations {
	if binding, exists := c.writers[nameID]; exists {
		c.writers[nameID] = outputBinding{writer: writer, formatter: binding.formatter}
	}
	return c
}
func (c *configurations) ChangeFormatter(nameID string, formatter Formatter) *configurations {
	if binding, exists := c.writers[nameID]; exists {
		c.writers[nameID] = outputBinding{writer: binding.writer, formatter: formatter}
	}
	return c
}

func (c *configurations) Writer(nameID string) (*Writer, *Formatter) {
	if binding, exists := c.writers[nameID]; exists {
		return &binding.writer, &binding.formatter
	}
	return nil, nil
}
func (c *configurations) Writers() map[string]outputBinding {
	return c.writers
}
