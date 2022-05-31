package driver

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// config holds settings for a single named config.
// The JSON tag name for a field is used both for JSON encoding and as
// a named variable.
type config struct {
	// Filename for file-based output formats, stdout by default.
	Output string `json:"-"`

	// Display options.
	SourcePath string `json:"-"`
	TrimPath   string `json:"-"`
}

// defaultConfig returns the default configuration values; it is unaffected by
// flags and interactive assignments.
func defaultConfig() config {
	return config{
		SourcePath: "./",
	}
}

// currentConfig holds the current configuration values; it is affected by
// flags and interactive assignments.
var currentCfg = defaultConfig()
var currentMu sync.Mutex

func currentConfig() config {
	currentMu.Lock()
	defer currentMu.Unlock()
	return currentCfg
}

func setCurrentConfig(cfg config) {
	currentMu.Lock()
	defer currentMu.Unlock()
	currentCfg = cfg
}

// configField contains metadata for a single configuration field.
type configField struct {
	name         string              // JSON field name/key in variables
	saved        bool                // Is field saved in settings?
	field        reflect.StructField // Field in config
	choices      []string            // Name Of variables in group
	defaultValue string              // Default value for this field.
}

var (
	configFields []configField // Precomputed metadata per config field

	// configFieldMap holds an entry for every config field as well as an
	// entry for every valid choice for a multi-choice field.
	configFieldMap map[string]configField
)

func init() {
	// Config names for fields that are not saved in settings and therefore
	// do not have a JSON name.
	notSaved := map[string]string{
		// Following fields are also not placed in URLs.
		"Output":     "output",
		"SourcePath": "source_path",
	}

	// choices holds the list of allowed values for config fields that can
	// take on one of a bounded set of values.
	choices := map[string][]string{
		"sort": {"cum", "flat"},
	}

	def := defaultConfig()
	configFieldMap = map[string]configField{}
	t := reflect.TypeOf(config{})
	for i, n := 0, t.NumField(); i < n; i++ {
		field := t.Field(i)
		js := strings.Split(field.Tag.Get("json"), ",")
		if len(js) == 0 {
			continue
		}
		// Get the configuration name for this field.
		name := js[0]
		if name == "-" {
			name = notSaved[field.Name]
			if name == "" {
				// Not a configurable field.
				continue
			}
		}
		f := configField{
			name:    name,
			saved:   (name == js[0]),
			field:   field,
			choices: choices[name],
		}
		f.defaultValue = def.get(f)
		configFields = append(configFields, f)
		configFieldMap[f.name] = f
		for _, choice := range f.choices {
			configFieldMap[choice] = f
		}
	}
}

// fieldPtr returns a pointer to the field identified by f in *cfg.
func (cfg *config) fieldPtr(f configField) interface{} {
	// reflect.ValueOf: converts to reflect.Value
	// Elem: dereferences cfg to make *cfg
	// FieldByIndex: fetches the field
	// Addr: takes address of field
	// Interface: converts back from reflect.Value to a regular value
	return reflect.ValueOf(cfg).Elem().FieldByIndex(f.field.Index).Addr().Interface()
}

// get returns the value of field f in cfg.
func (cfg *config) get(f configField) string {
	switch ptr := cfg.fieldPtr(f).(type) {
	case *string:
		return *ptr
	case *int:
		return fmt.Sprint(*ptr)
	case *float64:
		return fmt.Sprint(*ptr)
	case *bool:
		return fmt.Sprint(*ptr)
	}
	panic(fmt.Sprintf("unsupported config field type %v", f.field.Type))
}

// set sets the value of field f in cfg to value.
func (cfg *config) set(f configField, value string) error {
	switch ptr := cfg.fieldPtr(f).(type) {
	case *string:
		if len(f.choices) > 0 {
			// Verify that value is one of the allowed choices.
			for _, choice := range f.choices {
				if choice == value {
					*ptr = value
					return nil
				}
			}
			return fmt.Errorf("invalid %q value %q", f.name, value)
		}
		*ptr = value
	case *int:
		v, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		*ptr = v
	case *float64:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		*ptr = v
	case *bool:
		v, err := stringToBool(value)
		if err != nil {
			return err
		}
		*ptr = v
	default:
		panic(fmt.Sprintf("unsupported config field type %v", f.field.Type))
	}
	return nil
}

// isConfigurable returns true if name is either the name of a config field, or
// a valid value for a multi-choice config field.
func isConfigurable(name string) bool {
	_, ok := configFieldMap[name]
	return ok
}

// isBoolConfig returns true if name is either name of a boolean config field,
// or a valid value for a multi-choice config field.
func isBoolConfig(name string) bool {
	f, ok := configFieldMap[name]
	if !ok {
		return false
	}
	if name != f.name {
		return true // name must be one possible value for the field
	}
	var cfg config
	_, ok = cfg.fieldPtr(f).(*bool)
	return ok
}

// completeConfig returns the list of configurable names starting with prefix.
func completeConfig(prefix string) []string {
	var result []string
	for v := range configFieldMap {
		if strings.HasPrefix(v, prefix) {
			result = append(result, v)
		}
	}
	return result
}

// configure stores the name=value mapping into the current config, correctly
// handling the case when name identifies a particular choice in a field.
func configure(name, value string) error {
	currentMu.Lock()
	defer currentMu.Unlock()
	f, ok := configFieldMap[name]
	if !ok {
		return fmt.Errorf("unknown config field %q", name)
	}
	if f.name == name {
		return currentCfg.set(f, value)
	}
	// name must be one of the choices. If value is true, set field-value
	// to name.
	if v, err := strconv.ParseBool(value); v && err == nil {
		return currentCfg.set(f, name)
	}
	return fmt.Errorf("unknown config field %q", name)
}

// resetTransient sets all transient fields in *cfg to their currently
// configured values.
func (cfg *config) resetTransient() {
	current := currentConfig()
	cfg.Output = current.Output
	cfg.SourcePath = current.SourcePath
	cfg.TrimPath = current.TrimPath
}
