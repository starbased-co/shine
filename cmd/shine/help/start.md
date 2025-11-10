# shine start

Start the shinectl service manager and all enabled panels.

If shinectl is already running, this command reports success without taking action.

## USAGE
  shine start

## FLAGS
  --help   Show help for command

## BEHAVIOR
  When shinectl starts, it:
  - Reads configuration from ~/.config/shine/prism.toml
  - Spawns Kitty panels for each enabled prism via remote control
  - Launches prismctl supervisors to manage prism lifecycle
  - Creates IPC sockets in /run/user/{uid}/shine/ for communication

## EXAMPLES
  $ shine start
  $ shine start && shine status

## LEARN MORE
  Use `shine help status` to check running panels.
  Use `shine help logs` to view service logs.
  Config file: ~/.config/shine/prism.toml
