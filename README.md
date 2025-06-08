# ⏲️ Gomodoro
![photo_2025-06-08_12-35-48-removebg-preview](https://github.com/user-attachments/assets/9bdb38f8-423e-48c5-af2a-c5295197b644)

A lightning-fast Pomodoro timer for developers, built in Go.  

---

## 🚧 Project Status

> 🚧 Currently in development – some features are still being implemented.

---

## ❓ Why Gomodoro?

✔️ **Blazing fast** – native Go CLI performance  
✔️ **Minimalist design** – no clutter, just focus  
✔️ **Fully customizable** – tweak your work/break durations  
✔️ **Productivity stats** – track your streaks (coming soon)  
✔️ **Cross-platform** – works on macOS, Linux, and Windows

---

## ✨ Planned Features
- ⏱️ **Flexible timers** – Set custom durations with flags like `--work`, `--sbreak`, and `--lbreak`
- 🔁 **Auto-cycles** – Automatically take a long break after 4 work sessions
- 🔔 **Desktop notifications** – Visual alerts (e.g. `notify-send`) on timer completion
- 📟 **Theming** – Retro terminal progress bars with multiple themes

---

## 🛠️ Usage

```bash
$ gomodoro --work 25 --sbreak 5 --lbreak 15
```

Or simply use the defaults:

```bash
$ gomodoro
# Defaults: 25 min work / 5 min short break / 15 min long break
```

### ⚙️ Flags

| Flag       | Description                     | Default |
| ---------- | ------------------------------- | ------- |
| `--work`   | Work session duration (minutes) | 25      |
| `--sbreak` | Short break duration (minutes)  | 5       |
| `--lbreak` | Long break duration (minutes)   | 15      |

## 📦 Installation

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

- [ ] Desktop notifications
- [ ] Config file support (`~/.gomodoro.yaml`)
- [ ] Terminal color themes
- [ ] Sound alerts
- [ ] Infinite loop support (`--loop`)
- [ ] Pause/resume functionality
