package config

import (
	"testing"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// everyDangerousGlobal is the complete list of names lua.OpenBase adds
// that this sandbox must never expose, enumerated directly from
// yuin/gopher-lua's baselib.go rather than guessed. Any name here that
// survives (is non-nil) after newSandboxState is a real sandbox breach.
var everyDangerousGlobal = []string{
	"load", "loadfile", "loadstring", "dofile", "module", "newproxy",
	"getfenv", "setfenv", "print", "collectgarbage", "pcall", "xpcall",
	"rawget", "rawset", "rawequal", "unpack", "setmetatable", "getmetatable",
	"_G", "_printregs", "_VERSION", "_GOPHER_LUA_VERSION",
}

func TestSandboxDoesNotExposeDangerousGlobals(t *testing.T) {
	L, cancel := newSandboxState()
	defer cancel()
	defer L.Close()

	for _, name := range everyDangerousGlobal {
		t.Run(name, func(t *testing.T) {
			if v := L.GetGlobal(name); v != lua.LNil {
				t.Errorf("global %q is exposed (%v), it must not be", name, v)
			}
		})
	}
}

func TestSandboxDoesNotOpenOSIOOrDebug(t *testing.T) {
	L, cancel := newSandboxState()
	defer cancel()
	defer L.Close()

	for _, name := range []string{"os", "io", "debug", "package"} {
		t.Run(name, func(t *testing.T) {
			if v := L.GetGlobal(name); v != lua.LNil {
				t.Errorf("library %q is available (%v), it must never be opened", name, v)
			}
		})
	}
}

func TestSandboxStringLibraryHasNoDump(t *testing.T) {
	L, cancel := newSandboxState()
	defer cancel()
	defer L.Close()

	strTbl, ok := L.GetGlobal("string").(*lua.LTable)
	if !ok {
		t.Fatal("string library is not available")
	}
	if v := strTbl.RawGetString("dump"); v != lua.LNil {
		t.Errorf("string.dump is exposed (%v), it must be removed", v)
	}
}

func TestSandboxAllowsBasicTableStringMathOperations(t *testing.T) {
	L, cancel := newSandboxState()
	defer cancel()
	defer L.Close()

	err := L.DoString(`
		local sum = 0
		for i, v in ipairs({1, 2, 3}) do sum = sum + v end
		assert(sum == 6)
		assert(type({}) == "table")
		assert(string.upper("ok") == "OK")
		assert(math.floor(1.9) == 1)
		return true
	`)
	if err != nil {
		t.Fatalf("expected basic operations to work, got error: %v", err)
	}
}

func TestSandboxExecutionTimeoutRealyInterruptsAnInfiniteLoop(t *testing.T) {
	L, cancel := newSandboxStateWithTimeout(50 * time.Millisecond)
	defer cancel()
	defer L.Close()

	start := time.Now()
	err := L.DoString(`while true do end`)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected the infinite loop to be interrupted with an error")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("the loop was not really interrupted: took %v", elapsed)
	}
}
