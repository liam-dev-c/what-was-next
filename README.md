# what-was-next

A simple terminal task manager and time tracker — a small take on Super Productivity.

## Features

- Task list per project (add, edit, complete, delete, reorder)
- Per-task time tracking with a live timer
- Projects to group tasks
- Daily summary of completed tasks and time tracked

## Install

```bash
go install github.com/liam-dev-c/what-was-next@latest
```

## Usage

Run `what-was-next`. Data is stored at `~/.config/what-was-next/what-was-next.db`
(honoring `XDG_CONFIG_HOME`).

The daily summary groups tasks by your **local** calendar day. Timestamps are
stored in UTC and converted to your machine's timezone when computing "today".

### Keys

| Key | Action |
|-----|--------|
| `a` | add task/project |
| `e` | edit task |
| `enter` / `space` | toggle done / select |
| `d` | delete task |
| `J` / `K` | move task down / up |
| `t` | start/stop timer on task |
| `p` | projects |
| `s` | daily summary |
| `esc` | back |
| `q` | quit |
