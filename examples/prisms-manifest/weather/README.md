# Weather Prism with Manifest

This example demonstrates a prism using manifest-based discovery via `prism.toml`.

## Structure

```
weather/
├── prism.toml          # Manifest file (metadata)
├── shine-weather       # Binary (from examples/prisms/weather)
└── README.md          # This file
```

## Manifest Benefits

1. **Structured Metadata**: Version, author, license info
2. **Rich Discovery**: Tags, homepage, description
3. **Validation**: Automatic manifest validation

## Usage

### 1. Build the Weather Prism

```bash
cd examples/prisms/weather
make
```

### 2. Copy to Manifest Directory

```bash
mkdir -p ~/.config/shine/prisms/weather
cp examples/prisms-manifest/weather/prism.toml ~/.config/shine/prisms/weather/
cp examples/prisms/weather/shine-weather ~/.config/shine/prisms/weather/
```

### 3. Configure Shine

Edit `~/.config/shine/shine.toml`:

```toml
[core]
prism_dirs = ["~/.config/shine/prisms"]
discovery_mode = "manifest"  # or "auto"

[prisms.weather]
enabled = true
edge = "top-right"
columns_pixels = 300
lines_pixels = 80
```

### 4. Launch

```bash
shine
```

## Discovery Modes

- `convention`: Uses shine-* naming, ignores manifests
- `manifest`: Requires prism.toml, validates metadata
- `auto`: Tries manifest first, falls back to convention (recommended)

## Manifest Validation

The manifest is validated on load:
- Required fields: name, version, binary
- Binary path checked for existence and permissions

## Creating Your Own

1. Copy this directory structure
2. Edit `prism.toml` with your prism details
3. Build your prism binary
4. Place in user prism directory
5. Configure in `shine.toml`

## See Also

- `examples/prism.toml` - Full manifest format reference
- `docs/PRISM_SYSTEM_DESIGN.md` - Phase 3 documentation
- `examples/prisms/weather/` - Source code
