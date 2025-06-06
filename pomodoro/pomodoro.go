package pomodoro

const DEFAULT_WORK_TIMER = 25
const DEFAULT_SHORT_BREAK = 5
const DEFAULT_LONG_BREAK = 15

type Sleeper interface {
	Work()
	ShortBreak()
	LongBreak()
}

func Pomodoro(sleeper Sleeper, params ...int) {
	for range 4 {
		sleeper.Work()
		sleeper.ShortBreak()
	}

	sleeper.LongBreak()
}
