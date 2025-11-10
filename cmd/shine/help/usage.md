# shine - Prism TUI Manager

Manage TUI-based desktop shell panels for Hyprland using Kitty.

## USAGE
  shine <command> [flags]

## CORE COMMANDS
  start:      Start the shine service and enabled panels
  stop:       Stop all panels
  reload:     Reload configuration and update panels
  status:     Show status of all panels
  logs:       View service and panel logs

## ADDITIONAL COMMANDS
  help:       Show help for a command or topic
  version:    Show version information

## FLAGS
  --help      Show help for command
  --version   Show shine version

## EXAMPLES
  $ shine start
  $ shine status
  $ shine logs shinectl
  $ shine help start

## LEARN MORE
  Use `shine help <command>` for more information about a command.
  Read the manual at https://github.com/starbased-co/shine
  Config file: ~/.config/shine/prism.toml
