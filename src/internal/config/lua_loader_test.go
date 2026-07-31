package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// writeFiles creates each file (relative to dir) with the given content,
// creating parent directories as needed, and returns dir.
func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return dir
}

func TestLoadLuaConfigSingleFile(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"config.lua": `return {
			version = 1,
			timer = {work = "25m"},
		}`,
	})

	tbl, err := loadLuaConfig(filepath.Join(dir, "config.lua"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tbl.Type().String() != "table" {
		t.Fatalf("want a table, got %s", tbl.Type())
	}

	fc, err := tableToFileConfig(tbl, "config.lua")
	if err != nil {
		t.Fatalf("unexpected schema error: %v", err)
	}
	if fc.Timer == nil || *fc.Timer.Work != 25*time.Minute {
		t.Errorf("Timer.Work = %+v, want 25m", fc.Timer)
	}
}

func TestLoadLuaConfigModularRequire(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"config.lua": `
			local timer = require("settings.timer")
			local alerts = require("settings.alerts")
			local style = require("themes.default")
			return {version = 1, timer = timer, alerts = alerts, style = style}
		`,
		"settings/timer.lua":  `return {work = "25m", short_break = "5m", long_break = "15m"}`,
		"settings/alerts.lua": `return {notify = true, sound = true}`,
		"themes/default.lua":  `return {theme = "default"}`,
	})

	tbl, err := loadLuaConfig(filepath.Join(dir, "config.lua"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fc, err := tableToFileConfig(tbl, "config.lua")
	if err != nil {
		t.Fatalf("unexpected schema error: %v", err)
	}
	if fc.Timer == nil || fc.Timer.Work == nil || *fc.Timer.Work != 25*time.Minute {
		t.Errorf("Timer.Work not resolved correctly from the nested module: %+v", fc.Timer)
	}
	if fc.Style == nil || fc.Style.Theme == nil || *fc.Style.Theme != "default" {
		t.Errorf("Style.Theme not resolved correctly: %+v", fc.Style)
	}
}

func TestLoadLuaConfigMissingModule(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"config.lua": `return {version = 1, timer = require("settings.missing")}`,
	})

	_, err := loadLuaConfig(filepath.Join(dir, "config.lua"))
	if err == nil {
		t.Fatal("expected an error for a missing module")
	}
	if !strings.Contains(err.Error(), "settings.missing") && !strings.Contains(err.Error(), "settings") {
		t.Errorf("expected the error to name the missing module, got: %v", err)
	}
}

func TestLoadLuaConfigSyntaxErrorHasChunkAndLine(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"config.lua": "return {\n  version = ,\n}",
	})

	_, err := loadLuaConfig(filepath.Join(dir, "config.lua"))
	if err == nil {
		t.Fatal("expected a syntax error")
	}
	if !strings.Contains(err.Error(), "config.lua") {
		t.Errorf("expected the error to name the file, got: %v", err)
	}
	if !strings.Contains(err.Error(), "2") {
		t.Errorf("expected the error to reference line 2, got: %v", err)
	}
}

func TestLoadLuaConfigCircularDependency(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"config.lua": `return {version = 1, timer = require("a")}`,
		"a.lua":      `return require("b")`,
		"b.lua":      `return require("a")`,
	})

	_, err := loadLuaConfig(filepath.Join(dir, "config.lua"))
	if err == nil {
		t.Fatal("expected a circular dependency error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "a.lua") || !strings.Contains(msg, "b.lua") {
		t.Errorf("expected the error to show the full import chain (a.lua and b.lua), got: %v", msg)
	}
}

func TestLoadLuaConfigRejectsAbsolutePathModule(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"config.lua": `return {version = 1, timer = require("/etc/passwd")}`,
	})
	_, err := loadLuaConfig(filepath.Join(dir, "config.lua"))
	if err == nil {
		t.Fatal("expected an error for an absolute-path module name")
	}
}

func TestLoadLuaConfigRejectsParentTraversal(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"config.lua": `return {version = 1, timer = require("..secrets")}`,
	})
	_, err := loadLuaConfig(filepath.Join(dir, "config.lua"))
	if err == nil {
		t.Fatal("expected an error for a module name attempting to escape the root")
	}
}

func TestLoadLuaConfigMissingReturnValue(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"config.lua": `local x = 1`, // no return
	})
	_, err := loadLuaConfig(filepath.Join(dir, "config.lua"))
	if err == nil {
		t.Fatal("expected an error when the config file returns nothing")
	}
}

func TestLoadLuaConfigWrongReturnType(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"config.lua": `return "not a table"`,
	})
	_, err := loadLuaConfig(filepath.Join(dir, "config.lua"))
	if err == nil {
		t.Fatal("expected an error when the config file returns a non-table")
	}
}

func TestLoadLuaConfigTooManyModules(t *testing.T) {
	files := map[string]string{}
	var requires strings.Builder
	for i := 0; i < maxModules+5; i++ {
		name := "m" + strconv.Itoa(i)
		files[name+".lua"] = "return {}"
		requires.WriteString(`require("` + name + `")` + "\n")
	}
	files["config.lua"] = requires.String() + `return {version = 1}`

	dir := writeFiles(t, files)
	_, err := loadLuaConfig(filepath.Join(dir, "config.lua"))
	if err == nil {
		t.Fatal("expected an error for exceeding the module count limit")
	}
}

func TestLoadLuaConfigImportDepthExceeded(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < maxImportDepth+5; i++ {
		this := "d" + strconv.Itoa(i)
		next := "d" + strconv.Itoa(i+1)
		files[this+".lua"] = `return require("` + next + `")`
	}
	files["d"+strconv.Itoa(maxImportDepth+5)+".lua"] = "return {}"
	files["config.lua"] = `return {version = 1, timer = require("d0")}`

	dir := writeFiles(t, files)
	_, err := loadLuaConfig(filepath.Join(dir, "config.lua"))
	if err == nil {
		t.Fatal("expected an error for exceeding the import depth limit")
	}
}

func TestLoadLuaConfigCyclicTableDoesNotHang(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"config.lua": `
			local t = {}
			t.self = t
			return {version = 1, weird = t}
		`,
	})
	// Must return promptly (not hang) either way -- the exact error just
	// needs to be a clean rejection, since "weird" is not a known
	// top-level field. What matters here is termination.
	done := make(chan struct{})
	go func() {
		defer close(done)
		loadLuaConfig(filepath.Join(dir, "config.lua"))
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("loadLuaConfig hung on a cyclic table")
	}
}
