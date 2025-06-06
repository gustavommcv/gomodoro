package pomodoro

const DEFAULT_WORK_TIMER = 25
const DEFAULT_SHORT_BREAK = 5
const DEFAULT_LONG_BREAK = 15

type Sleeper interface {
	Work(minutes int)
	ShortBreak(minutes int)
	LongBreak(minutes int)
}

// args should be supplied like this:
// gomodoro start --work 25m --break 5m
func Pomodoro(sleeper Sleeper, args ...int) {
	for range 4 {
		sleeper.Work(DEFAULT_WORK_TIMER)
		sleeper.ShortBreak(DEFAULT_SHORT_BREAK)
	}

	sleeper.LongBreak(DEFAULT_LONG_BREAK)
}
