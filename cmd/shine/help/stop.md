# shine stop

Stop all panel supervisors and managed prisms.

Sends shutdown signal to all running prismctl panel supervisors via IPC. Each supervisor sends SIGTERM to managed prisms, waits 20ms for graceful shutdown, then sends SIGKILL if needed.

Note: This stops panels but not the shinectl service manager, which remains running.

## USAGE

```bash
shine stop
```

## FLAGS

```
--help   Show help for command
```

## EXAMPLES

```bash
$ shine stop
```

```bash
$ shine stop && shine status
```

## LEARN MORE
  Use `shine help start` to restart panels.
  Use `shine help status` to verify shutdown.
  IPC sockets: /run/user/{uid}/shine/
