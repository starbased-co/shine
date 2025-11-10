# shine logs

View log files from the shine service and panels.

Without arguments, lists all available log files. With a filename, displays the last 50 lines of that log.

## USAGE

```bash
shine logs              # List all log files
shine logs <filename>   # View specific log (last 50 lines)
```

## FLAGS

```
--help   Show help for command
```

## LOG FILES

```
shinectl.log              Service manager logs
prismctl-{component}.log  Panel supervisor logs
```

## EXAMPLES

```bash
$ shine logs
```

```bash
$ shine logs shinectl
```

```bash
$ shine logs prismctl-panel-0
```

## ADVANCED USAGE

Follow logs in real-time:

```bash
$ tail -f ~/.local/share/shine/logs/shinectl.log
```

Search logs:

```bash
$ grep ERROR ~/.local/share/shine/logs/*.log
```

```bash
$ grep "shine-clock" ~/.local/share/shine/logs/*.log
```

## LEARN MORE
  Use `shine help status` to view current panel state.
  Log directory: ~/.local/share/shine/logs/
  Config file: ~/.config/shine/prism.toml
