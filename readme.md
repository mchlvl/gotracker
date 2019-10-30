# Gotracker

Saves activity logs to `{user}/AppData/Roaming/TimeTrackerLogs`. Path currently not configurable.

Logs can be parsed with [gotracker-parser](https://github.com/mchlvl/gotracker-parser) CLI.

Early activity attributed to previous day - clock runs 04:00-28:00 (instead of 00:00-24:00).

## Build

```
go build
```

## Run

### Shell script (recommended)

```
run.sh
```

### From executable
IMPORTANT - on windows make sure to run from GitBash

```
./gotracker
```

or if executing in CMD/PowerShell, make sure to disable QuickEdit (see [reddit post](https://www.reddit.com/r/node/comments/d0ggmb/disable_quickedit_mode_on_windowscmd/))

