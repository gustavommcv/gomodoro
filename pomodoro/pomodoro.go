package pomodoro

const DEFAULT_WORK_TIMER int = 25
const DEFAULT_SHORT_BREAK int = 5
const DEFAULT_LONG_BREAK int = 15
const CICLES_PER_LAP int = 4

type Sleeper interface {
	Work(minutes int)
	ShortBreak(minutes int)
	LongBreak(minutes int)
}

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

	for actualCicle := 1; actualCicle <= CICLES_PER_LAP; actualCicle++ {
		sleeper.Work(work)
		if actualCicle == CICLES_PER_LAP {
			sleeper.LongBreak(longBreak)
		} else {
			sleeper.ShortBreak(shortBreak)
		}
	}
}
