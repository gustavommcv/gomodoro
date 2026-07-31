package config

import (
	"context"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// executionTimeout bounds how long any single config load (including all
// of its required modules) may spend executing Lua. This is enforced by
// gopher-lua's own context check inside the VM's instruction loop -- a
// genuine interruption of the running script, not a goroutine abandoned
// after a timeout expires (verified: an unbounded `while true do end`
// returns a "context deadline exceeded" error within this bound, not a
// process that keeps running in the background).
const executionTimeout = 500 * time.Millisecond

// dangerousBaseGlobals is every name lua.OpenBase registers that this
// sandbox must not expose, taken directly from gopher-lua's baselib.go.
// Removing this exact, complete list (rather than opening everything and
// hoping nothing dangerous survives) is what lets TestSandboxDoesNot
// ExposeDangerousGlobals prove none of them leaked through.
var dangerousBaseGlobals = []string{
	"load", "loadfile", "loadstring", "dofile", // code loading/execution
	"module", "newproxy", // module system / proxy tricks
	"getfenv", "setfenv", // Lua 5.1 environment manipulation (a known sandbox-escape vector)
	"print", "collectgarbage", // not needed for declarative config
	"pcall", "xpcall", // error recovery isn't needed for declarative config; keeping errors fatal is what we want
	"rawget", "rawset", "rawequal", "unpack", // not needed; smaller surface
	"setmetatable", "getmetatable", // not needed; removes a class of exotic proxy-table tricks
	"_G",                              // do not expose the global table itself
	"_printregs",                      // debug-only internal
	"_VERSION", "_GOPHER_LUA_VERSION", // not needed; nothing string identity should influence config
}

// newSandboxState creates an *lua.LState with only the standard-library
// surface Gomodoro's config format needs: tables, strings, numbers,
// booleans, basic iteration, and (once installed by the caller) a
// controlled require(). It never opens os, io, debug, or the package
// library at all -- they are not "removed", they are simply never added,
// which is the strongest guarantee available (see baseFuncs below for the
// base-library exception, which does have to open-then-curate because
// gopher-lua does not offer a way to open only some base functions).
func newSandboxState() (*lua.LState, context.CancelFunc) {
	return newSandboxStateWithTimeout(executionTimeout)
}

// newSandboxStateWithTimeout is newSandboxState with an explicit timeout,
// used directly by tests that need to observe the interruption behavior on
// a short, deterministic bound instead of the production default.
func newSandboxStateWithTimeout(timeout time.Duration) (*lua.LState, context.CancelFunc) {
	L := lua.NewState(lua.Options{
		SkipOpenLibs:        true,
		CallStackSize:       64,
		RegistryMaxSize:     8192,
		IncludeGoStackTrace: false,
	})

	// table, string, and math are opened wholesale: every function they
	// register is pure data manipulation with no I/O or OS capability
	// (verified by reading tablelib.go/stringlib.go/mathlib.go directly),
	// except string.dump, removed individually below.
	lua.OpenTable(L)
	lua.OpenString(L)
	lua.OpenMath(L)

	// base has no "open only some functions" option in gopher-lua, so it
	// is opened and then curated down to the exact allow-listed subset:
	// type, tostring, tonumber, pairs, ipairs, next, select, assert,
	// error. Every other name baseFuncs registers is explicitly nilled
	// out via dangerousBaseGlobals above, and a test enumerates that exact
	// list to prove none of them survive.
	lua.OpenBase(L)
	for _, name := range dangerousBaseGlobals {
		L.SetGlobal(name, lua.LNil)
	}

	if strTbl, ok := L.GetGlobal("string").(*lua.LTable); ok {
		strTbl.RawSetString("dump", lua.LNil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	L.SetContext(ctx)

	return L, cancel
}
