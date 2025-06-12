# Gomodoro
<img src="https://github.com/user-attachments/assets/9bdb38f8-423e-48c5-af2a-c5295197b644" width="200" alt="gomodoro-logo">

**A lightning-fast CLI Pomodoro timer for developers, built in Go.** 

## Project Status
> Currently in development – some features are still being implemented.

## Usage

```bash
# Usage example (note: 'start' command not implemented yet - omit it for now):
$ gomodoro start --work 25 --sbreak 5 --lbreak 15
```

Or simply use the defaults:

```bash
$ gomodoro start
# Defaults: 25 min work / 5 min short break / 15 min long break
```

## Flags

| Flag       | Description                     | Default |
| ---------- | ------------------------------- | ------- |
| `--work`   | Work session duration (minutes) | 25      |
| `--sbreak` | Short break duration (minutes)  | 5       |
| `--lbreak` | Long break duration (minutes)   | 15      |

## Installation

### Using `go install`

```bash
go install github.com/gustavommcv/gomodoro@latest
```

### Manual build

```bash
git clone https://github.com/gustavommcv/gomodoro.git
cd gomodoro
go build -o gomodoro
```

## 📌 Roadmap

- [ ] Interactive Desktop notifications
- [ ] Config file support (`~/.gomodoro.yaml`)
- [ ] Customizable Terminal color themes
- [ ] Sound alerts
- [ ] Infinite loop support (`--loop`)
- [ ] Lap count
- [ ] Pause/resume/skip functionality
