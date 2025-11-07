# Shine Panel Configuration Guide

This guide explains the refactored panel configuration system introduced in Phase 2.

## Overview

The configuration has been simplified from 4 size fields to 2, and "edge" has been renamed to "anchor" with enhanced positioning capabilities.

## Configuration Fields

### Size Fields (Simplified)

Instead of separate `lines`, `columns`, `lines_pixels`, `columns_pixels` fields, we now have:

- **`width`**: Width as integer (columns) or string with "px" (pixels)
- **`height`**: Height as integer (lines) or string with "px" (pixels)

**Examples:**

```toml
width = 80           # 80 terminal columns
width = "1200px"     # 1200 pixels
height = 24          # 24 terminal lines
height = "600px"     # 600 pixels
```

### Anchor Field (renamed from "edge")

The `anchor` field specifies where the panel is positioned on screen.

**Default:** `"center"` (changed from "top")

**Valid values:**

- `"top"` - Top edge
- `"bottom"` - Bottom edge
- `"left"` - Left edge
- `"right"` - Right edge
- `"center"` - Screen center
- `"center-sized"` - Center with sizing constraints
- `"none"` - No anchoring
- `"background"` - Background layer
- `"top-left"` - Top-left corner
- `"top-right"` - Top-right corner
- `"bottom-left"` - Bottom-left corner
- `"bottom-right"` - Bottom-right corner
- `"absolute"` - Absolute positioning (NEW)

### Position Field (NEW)

The `position` field specifies coordinates relative to the anchor point.

**Format:** `"x,y"` where each component can be:
- Integer (interpreted as columns/lines)
- String with "px" suffix (pixels)

**Coordinate systems by anchor:**

| Anchor | Origin Point | Coordinate System |
|--------|--------------|-------------------|
| `absolute` | Top-left (0,0) | Absolute from screen edges |
| `center` | Screen center (0,0) | Relative to center |
| `top` | Top-left of edge | x from left, y from top |
| `bottom` | Bottom-left of edge | x from left, y from bottom |
| `top-left` | Corner | Both from corner |
| `top-right` | Corner | x from right, y from top |
| `bottom-left` | Corner | x from left, y from bottom |
| `bottom-right` | Corner | Both from corner (negative moves inward) |

**Examples:**

```toml
# Absolute positioning: 100px from left, 200px from top
anchor = "absolute"
position = "100px,200px"

# Center with offset
anchor = "center"
position = "50px,0"  # 50px right of center

# Top-right corner
anchor = "top-right"
position = "0,0"  # Exactly at corner

# Bottom with horizontal offset
anchor = "bottom"
position = "100px,10px"  # 100px from left, 10px from bottom
```

### Margin Fields (Refinement Offsets)

Margin fields now serve as **refinement offsets** rather than direct positioning.

**New behavior:**

```
final_margin = position_calculated_margin + margin_refinement_from_config
```

**Example:**

```toml
anchor = "bottom"
position = "100px,50px"  # Base position
margin_left = 20         # Add 20px refinement
margin_bottom = -10      # Subtract 10px refinement

# Final: --margin-left=120 --margin-bottom=40
```

## Complete Configuration Examples

### Status Bar (Full Width Top)

```toml
[prisms.bar]
enabled = true
anchor = "top"
height = "30px"
width = 200          # Approximate full width in columns
position = "0,0"
output_name = "DP-2"
focus_policy = "not-allowed"
```

### Chat Panel (Bottom Center)

```toml
[prisms.chat]
enabled = true
anchor = "bottom"
height = 10
width = 80
position = "0,0"
margin_left = 10
margin_right = 10
margin_bottom = 10
output_name = "DP-2"
focus_policy = "on-demand"
hide_on_focus_loss = true
```

### Clock Widget (Top-Right Corner)

```toml
[prisms.clock]
enabled = true
anchor = "top-right"
width = "150px"
height = "30px"
position = "0,0"
output_name = "DP-2"
focus_policy = "not-allowed"
```

### System Info (Absolute Positioning)

```toml
[prisms.sysinfo]
enabled = true
anchor = "absolute"
width = "200px"
height = "100px"
position = "10px,40px"  # CSS-like absolute positioning
output_name = "DP-2"
focus_policy = "not-allowed"
```

### Spotify Player (Bottom with Custom Position)

```toml
[prisms.spotify]
enabled = true
anchor = "bottom"
width = "600px"
height = "120px"
position = "0,0"
margin_left = 10
margin_right = 10
margin_bottom = 10
output_name = "DP-2"
focus_policy = "on-demand"
hide_on_focus_loss = false
```

## Absolute Positioning Mode

The new `anchor = "absolute"` mode provides CSS-like absolute positioning:

- Origin: Top-left corner is (0,0)
- Coordinates: Direct pixel/column control
- No smart transformations
- Under the hood: Uses `--edge=center` in kitten CLI

**Use cases:**

- Precise pixel-perfect positioning
- Overlays and HUD elements
- Custom panel arrangements
- Avoiding anchor-based coordinate transformations

## Migration from Old Config

**Old format:**

```toml
edge = "top"
lines_pixels = 30
columns_pixels = 1200
margin_top = 0
```

**New format:**

```toml
anchor = "top"
height = "30px"
width = "1200px"
position = "0,0"
# margins now optional refinements
```

## Display Configuration

**CRITICAL:** All panels must specify `output_name = "DP-2"`. Using DP-1 will cause system failure.

**Default:** The system defaults to DP-2 if not specified, but explicit configuration is recommended.

## Translation to Kitten CLI

The configuration system automatically translates to `kitten panel` CLI arguments:

```toml
anchor = "bottom"
width = "600px"
height = "120px"
position = "100px,50px"
margin_left = 10
```

Becomes:

```bash
kitten panel \
  --edge=bottom \
  --columns=600px \
  --lines=120px \
  --margin-left=110 \
  --margin-bottom=50 \
  --output-name=DP-2 \
  ...
```

## Additional Fields

All other configuration fields remain unchanged:

- `focus_policy`: "not-allowed", "exclusive", "on-demand"
- `hide_on_focus_loss`: boolean
- `toggle_visibility`: boolean
- `enabled`: boolean

## Best Practices

1. **Use pixels for precise sizing:** `width = "600px"` rather than `width = 60`
2. **Choose appropriate anchor:** Use semantic anchors (top-right) over absolute when possible
3. **Minimal margins:** Use position for base placement, margins for fine-tuning only
4. **Always specify DP-2:** Never rely on defaults for output_name
5. **Test positioning:** Verify panel placement after configuration changes

## Troubleshooting

**Panel not appearing:**
- Check `enabled = true`
- Verify `output_name = "DP-2"`
- Ensure dimensions are reasonable

**Wrong position:**
- Verify anchor matches intended coordinate system
- Check position format (comma-separated, no spaces)
- Review margin refinements

**Size issues:**
- Confirm px suffix for pixel values
- Check monitor resolution with `hyprctl monitors`
- Verify width/height are positive integers

## Future Enhancements

Planned improvements:

- Percentage-based sizing: `width = "50%"`
- Named anchors: `anchor = "spotify-panel"`
- Dynamic positioning: `position = "center-offset"`
- Multi-monitor spanning: `output_name = ["DP-2", "DP-3"]`
