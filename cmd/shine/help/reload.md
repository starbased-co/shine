# shine reload

Reload configuration and update panels without stopping prisms.

Triggers configuration refresh in the shinectl service manager. Re-reads ~/.config/shine/prism.toml, starts newly enabled prisms, stops disabled ones, and updates panel geometry if changed.

⚠️  Currently requires manual SIGHUP signal. IPC reload coming soon.

## USAGE

```bash
shine reload
```

## FLAGS

```
--help   Show help for command
```

## CURRENT WORKAROUND

Until IPC reload is implemented:

```bash
$ pkill -HUP shinectl
```

```bash
$ shine status  # Verify changes
```

## WHAT RELOADS

- enabled flag (start/stop prisms)
- Panel geometry (origin, width, height)
- Restart policies
- Prism configuration files

## WHAT DOES NOT RELOAD

- Prism binaries (requires restart)
- Environment variables
- Core config (config_dir, data_dir, log_level)

## EXAMPLES

```bash
$ vim ~/.config/shine/prism.toml
```

```bash
$ pkill -HUP shinectl
```

```bash
$ shine status
```

## LEARN MORE
  Use `shine help start` to restart if reload fails.
  Use `shine help logs` to verify reload.
  Config file: ~/.config/shine/prism.toml
