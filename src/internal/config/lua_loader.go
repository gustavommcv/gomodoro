package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// Small, documented limits protecting config load from abuse or mistakes.
// None of these are meant to ever be hit by a reasonable hand-written
// config; they exist to bound the worst case (a huge file, a runaway
// module count, or a deeply nested require chain), not to constrain
// legitimate use.
const (
	maxFileSize    = 64 * 1024 // bytes per .lua file (config.lua or any module)
	maxModules     = 64        // distinct modules require()'d per load
	maxImportDepth = 16        // nested require() chain length
)

// moduleLoader implements Gomodoro's own require(): every module name is
// resolved relative to root, cached so a module runs at most once even if
// required from multiple places, and tracked so a circular require is
// detected and reported instead of recursing forever.
type moduleLoader struct {
	L       *lua.LState
	root    string
	cache   map[string]*lua.LTable
	loading map[string]bool
	chain   []string // current import chain, root-first
	count   int

	// lastErr is set the moment a Go-level error is detected (bad module
	// name, missing file, path escape, circular dependency, a required
	// module's own load failure). require() then raises a generic Lua
	// error purely to unwind the call stack; the caller reports lastErr
	// instead of whatever text Lua wraps around that generic raise, which
	// is what lets the reported error carry the exact chain and cause.
	lastErr error
}

func newModuleLoader(L *lua.LState, root string) *moduleLoader {
	return &moduleLoader{
		L:       L,
		root:    root,
		cache:   make(map[string]*lua.LTable),
		loading: make(map[string]bool),
	}
}

func (m *moduleLoader) install() {
	m.L.SetGlobal("require", m.L.NewFunction(m.require))
}

// fail records err (if nothing more specific has been recorded yet) and
// raises a Lua error to unwind out of the current require() call.
func (m *moduleLoader) fail(format string, args ...any) int {
	if m.lastErr == nil {
		m.lastErr = &configLoadError{
			chain: append([]string(nil), m.chain...),
			cause: fmt.Errorf(format, args...),
		}
	}
	m.L.RaiseError("gomodoro: configuration error")
	return 0
}

func (m *moduleLoader) require(L *lua.LState) int {
	name := L.CheckString(1)

	if err := validateModuleName(name); err != nil {
		return m.fail("%v", err)
	}

	if cached, ok := m.cache[name]; ok {
		L.Push(cached)
		return 1
	}

	if m.loading[name] {
		chain := append(append([]string(nil), m.chain...), moduleNameToRelPath(name))
		m.lastErr = &configLoadError{chain: chain, cause: fmt.Errorf("circular dependency")}
		L.RaiseError("gomodoro: configuration error")
		return 0
	}

	m.count++
	if m.count > maxModules {
		return m.fail("too many modules loaded (limit %d)", maxModules)
	}
	if len(m.chain) >= maxImportDepth {
		return m.fail("import depth exceeded (limit %d)", maxImportDepth)
	}

	relPath := moduleNameToRelPath(name)
	absPath, err := m.resolveWithinRoot(relPath)
	if err != nil {
		return m.fail("require(%q): %v", name, err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return m.fail("require(%q): module not found", name)
	}
	if len(data) > maxFileSize {
		return m.fail("require(%q): file too large (max %d bytes)", name, maxFileSize)
	}

	m.loading[name] = true
	m.chain = append(m.chain, relPath)
	tbl, evalErr := m.evalModule(relPath, data)
	m.chain = m.chain[:len(m.chain)-1]
	delete(m.loading, name)

	if evalErr != nil {
		if m.lastErr == nil {
			chain := append(append([]string(nil), m.chain...), relPath)
			m.lastErr = &configLoadError{chain: chain, cause: evalErr}
		}
		L.RaiseError("gomodoro: configuration error")
		return 0
	}

	m.cache[name] = tbl
	L.Push(tbl)
	return 1
}

// evalModule compiles and runs source under chunkName (used verbatim in
// any syntax/runtime error Lua reports), returning the single table it
// must return.
func (m *moduleLoader) evalModule(chunkName string, data []byte) (*lua.LTable, error) {
	fn, err := m.L.Load(bytes.NewReader(data), chunkName)
	if err != nil {
		return nil, err
	}
	m.L.Push(fn)
	if err := m.L.PCall(0, 1, nil); err != nil {
		return nil, err
	}
	ret := m.L.Get(-1)
	m.L.Pop(1)

	tbl, ok := ret.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("%s: must return a table, got %s", chunkName, ret.Type().String())
	}
	return tbl, nil
}

// resolveWithinRoot turns a relative module path into an absolute path,
// refusing to return anything outside root -- including via a symlink
// (inside root) that points outside it. Module names are already
// restricted to a safe character set by validateModuleName before this is
// called, so this is defense in depth, not the only line of defense.
func (m *moduleLoader) resolveWithinRoot(relPath string) (string, error) {
	candidate := filepath.Join(m.root, relPath)

	realRoot, err := filepath.EvalSymlinks(m.root)
	if err != nil {
		realRoot = m.root
	}

	target := candidate
	if resolved, err := filepath.EvalSymlinks(candidate); err == nil {
		target = resolved
	}
	// If EvalSymlinks failed above, the file likely doesn't exist yet;
	// that surfaces as "module not found" from the ReadFile that follows.

	rel, err := filepath.Rel(realRoot, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("module path escapes the configuration root")
	}

	return candidate, nil
}

// configLoadError formats a config load failure as Gomodoro documents it:
// a header, the full chain of files that were being loaded when the
// failure happened, and the specific cause (which already has its own
// file:line prefix when the cause is a genuine Lua syntax/runtime error).
type configLoadError struct {
	chain []string
	cause error
}

func (e *configLoadError) Error() string {
	var b strings.Builder
	b.WriteString("failed to load Gomodoro configuration:\n")
	b.WriteString(strings.Join(e.chain, " -> "))
	b.WriteString("\n")
	b.WriteString(e.cause.Error())
	return b.String()
}

func (e *configLoadError) Unwrap() error { return e.cause }

// loadLuaConfig reads and evaluates path as the root Gomodoro Lua config
// file. require() calls within it (and within any module it requires) are
// resolved relative to path's own directory, which becomes the sandbox's
// module root -- nothing outside that directory is ever readable, and the
// Lua state is closed before this returns (nothing from the runtime stays
// alive afterward).
func loadLuaConfig(path string) (*lua.LTable, error) {
	root := filepath.Dir(path)
	chunkName := filepath.Base(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) > maxFileSize {
		return nil, fmt.Errorf("%s: file too large (max %d bytes)", chunkName, maxFileSize)
	}

	L, cancel := newSandboxState()
	defer cancel()
	defer L.Close()

	ml := newModuleLoader(L, root)
	ml.install()
	ml.chain = []string{chunkName}

	tbl, err := ml.evalModule(chunkName, data)
	if err != nil {
		if ml.lastErr != nil {
			return nil, ml.lastErr
		}
		return nil, &configLoadError{chain: []string{chunkName}, cause: err}
	}

	return tbl, nil
}
