# what-was-next

A simple terminal task manager and time tracker — a small take on Super Productivity.

## Features

- Task list per project (add, edit, complete, delete, reorder)
- Per-task time tracking with a live timer
- Projects to group tasks
- Summary of completed tasks and time tracked — the landing screen — with a
  day/week toggle (weekly view includes a per-day breakdown)
- Settings, including which day the week starts on

## Install

```bash
go install github.com/liam-dev-c/what-was-next@latest
```

## Usage

Run `what-was-next`. Data is stored at `~/.config/what-was-next/what-was-next.db`
(honoring `XDG_CONFIG_HOME`).

The summary opens by default. It groups tasks by your **local** calendar day (or
week). Timestamps are stored in UTC and converted to your machine's timezone
when computing "today" and "this week". The week starts on the day chosen in
settings (Monday by default).

### Keys

Tasks screen:

| Key | Action |
|-----|--------|
| `a` | add task/project |
| `e` | edit task |
| `enter` / `space` | toggle done / select |
| `d` | delete task |
| `J` / `K` | move task down / up |
| `t` | start/stop timer on task |
| `p` | projects |
| `s` | summary |
| `,` | settings |
| `esc` | back |
| `q` | quit |

Summary screen:

| Key | Action |
|-----|--------|
| `d` / `w` | day / week view |
| `t` | tasks |
| `p` | projects |
| `,` | settings |
| `q` | quit |
