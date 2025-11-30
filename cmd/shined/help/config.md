# shined config

Configuration file format and loading behavior.

## USAGE

```bash
shined -config ~/.config/shine/shine.toml
```

## DEFAULT LOCATION

```text
~/.config/shine/shine.toml
```

## CONFIGURATION FORMAT

Example shine.toml:

```toml
[core]
log_level = "info"

[[prisms]]
name = "shine-clock"
enabled = true
origin = "top-right"
width = "200px"
height = "100px"

[[prisms]]
name = "shine-chat"
enabled = true
origin = "bottom-left"
width = "400px"
height = "300px"
```

## VALIDATION

shined validates configuration on startup and reload:

- Prism name must not be empty
- Origin must be valid (top-left, top-right, bottom-left, bottom-right)
- Dimensions must be valid (pixels or percentages)

Invalid configurations cause shined to exit or abort reload.

## HOT-RELOAD

```bash
pkill -HUP shined
```

Configuration reload does NOT restart existing panels.
Only adds/removes panels based on config changes.

## EXAMPLES

```bash
shined -config ~/.config/shine/dev-config.toml
```

```bash
vim ~/.config/shine/shine.toml
pkill -HUP shined
```

## LEARN MORE

Use `shined help signals` for reload behavior.
See shine.toml documentation for full reference.
Use `shine status` to verify configuration.
