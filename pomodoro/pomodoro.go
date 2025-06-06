package pomodoro

const DEFAULT_WORK_TIMER int = 25
const DEFAULT_SHORT_BREAK int  = 5
const DEFAULT_LONG_BREAK int = 15

type Sleeper interface {
	Work(minutes int)
	ShortBreak(minutes int)
	LongBreak(minutes int)
}

// args should be supplied like this:
// gomodoro start --work 25m --break 5m
func Pomodoro(sleeper Sleeper, work, shortBreak, longBreak int) {
	if work == 0 {
		work = DEFAULT_WORK_TIMER
	}

	if shortBreak == 0 {
		shortBreak = DEFAULT_SHORT_BREAK
	}

	if longBreak == 0 {
		longBreak = DEFAULT_LONG_BREAK
	}

	for range 4 {
		sleeper.Work(work)
		sleeper.ShortBreak(shortBreak)
	}

	sleeper.LongBreak(longBreak)
}
