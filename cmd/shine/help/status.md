# shine status

Display current state of all panel supervisors and their managed prisms.

Queries all running prismctl panel supervisors via IPC to show which panel is running, foreground prism (actively displayed), background prisms (suspended), and per-prism state.

## USAGE

```bash
shine status
```

## FLAGS

```
--help   Show help for command
```

## OUTPUT

For each panel:
- Panel name and IPC socket path
- Foreground prism (currently visible, receiving input)
- Background prisms (suspended with SIGSTOP, not consuming CPU)
- Table with prism name, PID, and state

## EXAMPLES

```bash
$ shine status
```

```bash
$ shine start && shine status
```

## LEARN MORE
  Use `shine help start` to start panels.
  Use `shine help logs` to debug status issues.
  IPC sockets: /run/user/{uid}/shine/
