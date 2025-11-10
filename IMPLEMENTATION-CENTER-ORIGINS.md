# Implementation: Center and Center-Sized Origins

## Summary

Implemented support for both `center` and `center-sized` origin options to properly handle Kitty's edge system constraints.

## Problem Solved

### Critical Bug in `origin = "center"`

**Previous behavior**: `OriginCenter` mapped to Kitty's `edge=center`, but only set `top` and `left` margins.

**Kitty's `edge=center` behavior**:
- Anchors panel to ALL four sides (top, left, bottom, right)
- Panel is shrunk and placed using margin parameters
- Requires ALL FOUR margins to be set correctly for proper centering

**Bug**: Panel would extend to screen edges instead of being properly sized and centered.

## Implementation Details

### 1. New Origin Type: `OriginCenterSized`

Added new enum value to `pkg/panel/config.go`:

```go
const (
    OriginTopLeft Origin = iota
    OriginTopCenter
    OriginTopRight
    OriginLeftCenter
    OriginCenter
    OriginCenterSized  // NEW: Auto-centers with no margin calculations
    OriginRightCenter
    OriginBottomLeft
    OriginBottomCenter
    OriginBottomRight
)
```

### 2. String Conversions

Updated `Origin.String()` and `ParseOrigin()`:
- `OriginCenter.String()` → `"center"`
- `OriginCenterSized.String()` → `"center-sized"`
- `ParseOrigin("center")` → `OriginCenter`
- `ParseOrigin("center-sized")` → `OriginCenterSized`

### 3. Edge Mapping

Updated `originToEdge()` to return correct Kitty edge values:
- `OriginCenter` → `"center"` (edge=center)
- `OriginCenterSized` → `"center-sized"` (edge=center-sized)

### 4. Fixed Margin Calculations

#### `origin = "center"` (Fixed)

**Critical fix**: Now sets ALL FOUR margins when using `edge=center`:

```go
case OriginCenter:
    // CRITICAL: edge=center anchors to ALL sides, so we MUST set all four margins
    left = (monWidth / 2) - (panelWidth / 2) + offsetX
    top = (monHeight / 2) - (panelHeight / 2) + offsetY
    right = (monWidth / 2) - (panelWidth / 2) - offsetX   // NEW
    bottom = (monHeight / 2) - (panelHeight / 2) - offsetY // NEW
```

**Behavior**:
- Supports position offsets via `position = "x,y"`
- All four margins calculated to properly center and size the panel
- Asymmetric margins when using position offsets

#### `origin = "center-sized"` (New)

**Early return**: Auto-centering, no margin calculations needed:

```go
if c.Origin == OriginCenterSized {
    return 0, 0, 0, 0, nil
}
```

**Behavior**:
- Maps to Kitty `edge=center-sized`
- Auto-centers panel, no margin args generated
- Position offsets are ignored (margins have no effect)
- Simpler, recommended for basic centered panels

### 5. Configuration Examples

Added to `examples/shine.toml`:

```toml
# Simple centered panel (auto-centers)
[prisms.sysinfo]
enabled = false
origin = "center-sized"
width = "200px"
height = "100px"
# NOTE: position is ignored for center-sized

# Centered panel with position offset
[prisms.notification]
enabled = false
origin = "center"
width = "400px"
height = "300px"
position = "50,100"  # 50px right, 100px down from center
```

### 6. Comprehensive Tests

Added tests to `pkg/panel/config_test.go`:

**`TestOriginCenterSized`**:
- String conversion
- Parse from string
- `originToEdge()` returns `"center-sized"`
- `calculateMargins()` returns all zeros
- `ToRemoteControlArgs()` omits margin args

**`TestOriginCenterFourMargins`**:
- `calculateMargins()` without offset (symmetric margins)
- `calculateMargins()` with offset (asymmetric margins)
- `ToRemoteControlArgs()` includes all four margin args

## Generated Kitty Args

### center-sized (Auto-centering)

```bash
kitty @ launch \
  --type=os-panel \
  --os-panel edge=center-sized \
  --os-panel columns=400px \
  --os-panel lines=300px \
  --os-panel output-name=DP-2 \
  /usr/bin/prism
```

**No margin args** - Kitty auto-centers the panel.

### center (With all four margins)

```bash
kitty @ launch \
  --type=os-panel \
  --os-panel edge=center \
  --os-panel columns=400px \
  --os-panel lines=300px \
  --os-panel margin-top=570 \
  --os-panel margin-left=1080 \
  --os-panel margin-bottom=570 \
  --os-panel margin-right=1080 \
  --os-panel output-name=DP-2 \
  /usr/bin/prism
```

**All four margins set** - Kitty shrinks panel and places using margins.

### center with offset (Asymmetric margins)

```bash
kitty @ launch \
  --type=os-panel \
  --os-panel edge=center \
  --os-panel columns=400px \
  --os-panel lines=300px \
  --os-panel margin-top=670 \
  --os-panel margin-left=1130 \
  --os-panel margin-bottom=470 \
  --os-panel margin-right=1030 \
  --os-panel output-name=DP-2 \
  /usr/bin/prism
```

**Asymmetric margins** - Offset 50px right, 100px down from center.

## Testing Results

All tests pass:

```bash
$ go test ./pkg/panel -v
=== RUN   TestOriginCenterSized
=== RUN   TestOriginCenterSized/String_conversion
=== RUN   TestOriginCenterSized/Parse_from_string
=== RUN   TestOriginCenterSized/originToEdge
=== RUN   TestOriginCenterSized/calculateMargins_returns_zero
=== RUN   TestOriginCenterSized/ToRemoteControlArgs_omits_margin_args
--- PASS: TestOriginCenterSized (0.00s)

=== RUN   TestOriginCenterFourMargins
=== RUN   TestOriginCenterFourMargins/calculateMargins_without_offset
=== RUN   TestOriginCenterFourMargins/calculateMargins_with_offset
=== RUN   TestOriginCenterFourMargins/ToRemoteControlArgs_includes_all_four_margins
--- PASS: TestOriginCenterFourMargins (0.03s)

PASS
ok  	github.com/starbased-co/shine/pkg/panel	0.135s
```

## Usage Recommendations

**Use `origin = "center-sized"`** when:
- You want a simple centered panel
- No position offsets needed
- Simpler configuration

**Use `origin = "center"`** when:
- You need position offsets from center
- You want precise control over centering
- You need asymmetric positioning

## Backward Compatibility

✅ Existing configs with `origin = "center"` still work, but now properly set all four margins
✅ No breaking changes to API
✅ All existing tests pass

## Files Modified

1. `pkg/panel/config.go`:
   - Added `OriginCenterSized` enum value
   - Updated `Origin.String()` method
   - Updated `ParseOrigin()` function
   - Updated `originToEdge()` method
   - **Fixed** `calculateMargins()` for `OriginCenter` (all four margins)
   - Added `calculateMargins()` early return for `OriginCenterSized`

2. `pkg/panel/config_test.go`:
   - Added `TestOriginCenterSized` test suite
   - Added `TestOriginCenterFourMargins` test suite
   - Updated `TestParseOrigin` to include "center-sized"
   - Updated `TestOriginString` to include `OriginCenterSized`

3. `examples/shine.toml`:
   - Updated `[prisms.sysinfo]` to use `origin = "center-sized"`
   - Added commented example for `origin = "center"` with offset
   - Added detailed documentation explaining differences

## Date

2025-11-09
