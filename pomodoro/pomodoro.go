package pomodoro

const DEFAULT_WORK_TIMER int = 25
const DEFAULT_SHORT_BREAK int = 5
const DEFAULT_LONG_BREAK int = 15
const CICLES_PER_LAP int = 4
const work string = "work"
const shortBreak string = "shortBreak"
const longBreak string = "longBreak"

type Sleeper interface {
	Work(minutes int)
	ShortBreak(minutes int)
	LongBreak(minutes int)
}

type PomodoroOperations struct {
	Calls []string
}

func (p *PomodoroOperations) Work() {
	p.Calls = append(p.Calls, work)
}
func (p *PomodoroOperations) shortBreak() {
	p.Calls = append(p.Calls, shortBreak)
}
func (p *PomodoroOperations) longBreak() {
	p.Calls = append(p.Calls, longBreak)
}

// args should be supplied like this:
// gomodoro start --work 25m --break 5m
func Pomodoro(operation *PomodoroOperations, sleeper Sleeper, work, shortBreak, longBreak int) {
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
		operation.Work()
		if actualCicle == CICLES_PER_LAP {
			sleeper.LongBreak(longBreak)
			operation.longBreak()
		} else {
			sleeper.ShortBreak(shortBreak)
			operation.shortBreak()
		}
	}

}
