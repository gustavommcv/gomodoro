package config

import (
	"fmt"
	"math"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// supportedSchemaVersion is the highest config schema version this build
// of Gomodoro understands. version is a required field (there is no safe
// default to assume for it), and any version above this is rejected with
// a migration hint rather than silently ignored or partially applied.
const supportedSchemaVersion = 1

var (
	knownTopLevelFields = []string{"version", "timer", "alerts", "style"}
	knownTimerFields    = []string{"work", "short_break", "long_break"}
	knownAlertFields    = []string{"notify", "sound"}
	knownStyleFields    = []string{"theme"}
)

// tableToFileConfig converts and validates the root table a Gomodoro Lua
// config file returned into a FileConfig. chunkName identifies the file
// the table came from, for error messages. Validation is all-or-nothing:
// the first problem found is returned as an error and nothing from a
// partially-valid table is ever applied.
func tableToFileConfig(tbl *lua.LTable, chunkName string) (FileConfig, error) {
	if err := checkUnknownFields(tbl, "", knownTopLevelFields, chunkName); err != nil {
		return FileConfig{}, err
	}

	version, err := requiredVersion(tbl, chunkName)
	if err != nil {
		return FileConfig{}, err
	}

	fc := FileConfig{Version: version}

	if v := tbl.RawGetString("timer"); v != lua.LNil {
		t, ok := v.(*lua.LTable)
		if !ok {
			return FileConfig{}, fieldError(chunkName, "timer", fmt.Sprintf("must be a table, got %s", v.Type()))
		}
		tc, err := timerFromTable(t, chunkName)
		if err != nil {
			return FileConfig{}, err
		}
		fc.Timer = tc
	}

	if v := tbl.RawGetString("alerts"); v != lua.LNil {
		t, ok := v.(*lua.LTable)
		if !ok {
			return FileConfig{}, fieldError(chunkName, "alerts", fmt.Sprintf("must be a table, got %s", v.Type()))
		}
		ac, err := alertsFromTable(t, chunkName)
		if err != nil {
			return FileConfig{}, err
		}
		fc.Alerts = ac
	}

	if v := tbl.RawGetString("style"); v != lua.LNil {
		t, ok := v.(*lua.LTable)
		if !ok {
			return FileConfig{}, fieldError(chunkName, "style", fmt.Sprintf("must be a table, got %s", v.Type()))
		}
		sc, err := styleFromTable(t, chunkName)
		if err != nil {
			return FileConfig{}, err
		}
		fc.Style = sc
	}

	return fc, nil
}

func requiredVersion(tbl *lua.LTable, chunkName string) (int, error) {
	v := tbl.RawGetString("version")
	if v == lua.LNil {
		return 0, fieldError(chunkName, "version", "is required (e.g. version = 1)")
	}
	num, ok := v.(lua.LNumber)
	if !ok {
		return 0, fieldError(chunkName, "version", fmt.Sprintf("must be a number, got %s", v.Type()))
	}
	f := float64(num)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fieldError(chunkName, "version", "must be a finite number")
	}
	if f != math.Trunc(f) {
		return 0, fieldError(chunkName, "version", "must be a whole number")
	}
	version := int(f)
	if version < 1 {
		return 0, fieldError(chunkName, "version", "must be >= 1")
	}
	if version > supportedSchemaVersion {
		return 0, fieldError(chunkName, "version",
			fmt.Sprintf("version %d is newer than this build of Gomodoro supports (max %d) -- upgrade Gomodoro, or lower the config's version if it was set by mistake", version, supportedSchemaVersion))
	}
	return version, nil
}

func timerFromTable(t *lua.LTable, chunkName string) (*FileTimerConfig, error) {
	if err := checkUnknownFields(t, "timer", knownTimerFields, chunkName); err != nil {
		return nil, err
	}
	tc := &FileTimerConfig{}

	if v := t.RawGetString("work"); v != lua.LNil {
		d, err := parseDurationField(v, chunkName, "timer.work")
		if err != nil {
			return nil, err
		}
		tc.Work = &d
	}
	if v := t.RawGetString("short_break"); v != lua.LNil {
		d, err := parseDurationField(v, chunkName, "timer.short_break")
		if err != nil {
			return nil, err
		}
		tc.ShortBreak = &d
	}
	if v := t.RawGetString("long_break"); v != lua.LNil {
		d, err := parseDurationField(v, chunkName, "timer.long_break")
		if err != nil {
			return nil, err
		}
		tc.LongBreak = &d
	}
	return tc, nil
}

func alertsFromTable(t *lua.LTable, chunkName string) (*FileAlertConfig, error) {
	if err := checkUnknownFields(t, "alerts", knownAlertFields, chunkName); err != nil {
		return nil, err
	}
	ac := &FileAlertConfig{}

	if v := t.RawGetString("notify"); v != lua.LNil {
		b, ok := v.(lua.LBool)
		if !ok {
			return nil, fieldError(chunkName, "alerts.notify", fmt.Sprintf("must be a boolean, got %s", v.Type()))
		}
		val := bool(b)
		ac.Notify = &val
	}
	if v := t.RawGetString("sound"); v != lua.LNil {
		b, ok := v.(lua.LBool)
		if !ok {
			return nil, fieldError(chunkName, "alerts.sound", fmt.Sprintf("must be a boolean, got %s", v.Type()))
		}
		val := bool(b)
		ac.Sound = &val
	}
	return ac, nil
}

func styleFromTable(t *lua.LTable, chunkName string) (*FileStyleConfig, error) {
	if err := checkUnknownFields(t, "style", knownStyleFields, chunkName); err != nil {
		return nil, err
	}
	sc := &FileStyleConfig{}

	if v := t.RawGetString("theme"); v != lua.LNil {
		s, ok := v.(lua.LString)
		if !ok {
			return nil, fieldError(chunkName, "style.theme", fmt.Sprintf("must be a string, got %s", v.Type()))
		}
		if string(s) == "" {
			return nil, fieldError(chunkName, "style.theme", "must not be empty")
		}
		val := string(s)
		sc.Theme = &val
	}
	return sc, nil
}

// parseDurationField requires a non-empty, positive, whole-minute
// time.ParseDuration string (e.g. "25m"). Whole minutes specifically --
// not just "positive" -- because the timer engine itself only operates in
// whole minutes; a config asking for "25m30s" would silently lose the 30
// seconds if allowed through, so it is rejected instead.
func parseDurationField(v lua.LValue, chunkName, path string) (time.Duration, error) {
	s, ok := v.(lua.LString)
	if !ok {
		return 0, fieldError(chunkName, path, fmt.Sprintf("must be a duration string (e.g. \"25m\"), got %s", v.Type()))
	}
	str := string(s)
	if str == "" {
		return 0, fieldError(chunkName, path, "must not be empty")
	}
	d, err := time.ParseDuration(str)
	if err != nil {
		return 0, fieldError(chunkName, path, fmt.Sprintf("is not a valid duration: %v", err))
	}
	if d <= 0 {
		return 0, fieldError(chunkName, path, "must be a valid positive duration")
	}
	if d%time.Minute != 0 {
		return 0, fieldError(chunkName, path, fmt.Sprintf("must be a whole number of minutes (got %s)", d))
	}
	return d, nil
}

// checkUnknownFields reports the first key in t that isn't in known,
// including a "did you mean" suggestion when a close match exists.
func checkUnknownFields(t *lua.LTable, sectionPrefix string, known []string, chunkName string) error {
	var unknownKey string
	t.ForEach(func(k, _ lua.LValue) {
		if unknownKey != "" {
			return
		}
		key, ok := k.(lua.LString)
		if !ok {
			unknownKey = fmt.Sprintf("<%s key>", k.Type())
			return
		}
		name := string(key)
		for _, kn := range known {
			if kn == name {
				return
			}
		}
		unknownKey = name
	})
	if unknownKey == "" {
		return nil
	}

	path := unknownKey
	if sectionPrefix != "" {
		path = sectionPrefix + "." + unknownKey
	}
	msg := "is unknown"
	if suggestion := closestField(unknownKey, known); suggestion != "" {
		msg += fmt.Sprintf("; did you mean %q?", suggestion)
	}
	return fieldError(chunkName, path, msg)
}

func fieldError(chunkName, path, msg string) error {
	if path == "" {
		return fmt.Errorf("%s: %s", chunkName, msg)
	}
	return fmt.Errorf("%s: field %q %s", chunkName, path, msg)
}

// closestField returns the candidate closest to name by edit distance, if
// it's close enough to plausibly be a typo (distance <= 2), else "".
func closestField(name string, candidates []string) string {
	best := ""
	bestDist := -1
	for _, c := range candidates {
		d := levenshtein(name, c)
		if d <= 2 && (bestDist == -1 || d < bestDist) {
			best, bestDist = c, d
		}
	}
	return best
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	dp := make([]int, lb+1)
	for j := range dp {
		dp[j] = j
	}
	for i := 1; i <= la; i++ {
		prev := dp[0]
		dp[0] = i
		for j := 1; j <= lb; j++ {
			tmp := dp[j]
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			dp[j] = minInt(dp[j]+1, minInt(dp[j-1]+1, prev+cost))
			prev = tmp
		}
	}
	return dp[lb]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
