# GitHub CLI Help Style Guide

Documentation of `gh` CLI help patterns for reference when designing help pages.

## Core Principles

1. **Concise, not comprehensive** - Quick reference, not tutorials
2. **Scannable** - Users should understand command in seconds
3. **Traditional CLI** - Match Unix/POSIX conventions
4. **Consistent structure** - Same sections in same order

## Opening Line Format

**Pattern**: Single imperative sentence describing what the command does.

**Examples**:
- `gh`: "Work seamlessly with GitHub from the command line."
- `gh repo`: "Work with GitHub repositories."
- `gh repo clone`: "Clone a GitHub repository locally."
- `gh issue create`: "Create an issue on GitHub."

**Length**: 1-2 sentences maximum
**Tone**: Imperative, active voice
**Focus**: What it does, not why or how

## Section Structure

### Standard Order

1. **Title + Opening Line**
2. **USAGE** - Exact syntax
3. **Content Sections** (varies by command level)
   - Main: Command groups
   - Group: Command list
   - Command: Flags, behavior
4. **FLAGS** - Command options
5. **EXAMPLES** - Usage examples (3-5 max)
6. **LEARN MORE** - Standard footer

### Main Help (e.g., `gh --help`)

```
Title + opening line
USAGE
CORE COMMANDS (grouped)
{OTHER COMMAND GROUPS}
ADDITIONAL COMMANDS
HELP TOPICS
FLAGS
EXAMPLES
LEARN MORE
```

### Command Group Help (e.g., `gh repo --help`)

```
Title + opening line
USAGE
GENERAL COMMANDS
TARGETED COMMANDS
INHERITED FLAGS
ARGUMENTS (if applicable)
EXAMPLES
LEARN MORE
```

### Specific Command Help (e.g., `gh repo clone --help`)

```
Title + opening line
Additional context (1-2 paragraphs max)
USAGE
FLAGS
INHERITED FLAGS
EXAMPLES
LEARN MORE
```

## Section Guidelines

### USAGE

**Format**: Indented, exact syntax
```
USAGE
  gh <command> <subcommand> [flags]
```

**Rules**:
- Show exact invocation
- Use angle brackets `<>` for required args
- Use square brackets `[]` for optional args
- Use `...` for variadic args
- Keep to 1-2 lines

### Command Lists

**Format**: Two-column layout with alignment
```
CORE COMMANDS
  auth:          Authenticate gh and git with GitHub
  browse:        Open repositories, issues, and more
  issue:         Manage issues
```

**Rules**:
- Command name + colon + spaces + description
- Description is 1 line, verb-object format
- Group related commands under headers

### FLAGS

**Format**: Short flag, long flag, description
```
FLAGS
  -a, --assignee login   Assign people by their login
  -b, --body string      Supply a body
      --help             Show help for command
```

**Rules**:
- Align flag descriptions
- Keep descriptions to 1 line
- No verbose explanations
- Group inherited flags separately

### EXAMPLES

**Format**: Shell command with optional brief comment
```
EXAMPLES
  $ gh issue create
  $ gh repo clone cli/cli

  # Clone with custom directory
  $ gh repo clone cli/cli workspace/cli
```

**Rules**:
- 3-5 examples maximum
- Start with `$` prompt
- Minimal or no commentary
- Show common use cases only
- Group related examples with blank line

### LEARN MORE

**Format**: Standard footer with 3-4 lines
```
LEARN MORE
  Use `gh <command> <subcommand> --help` for more information about a command.
  Read the manual at https://cli.github.com/manual
  Learn about exit codes using `gh help exit-codes`
```

**Rules**:
- Always include next-level help
- Link to external docs
- Point to related help topics
- 3-4 lines maximum

## Description Style

### Length
- **Main help**: 1 sentence
- **Command group**: 1 sentence
- **Specific command**: 1-2 sentences + optional 1-2 paragraph context

### Tone
- **Imperative**: "Clone a repository" not "This command clones"
- **Active voice**: "Manage issues" not "Issues can be managed"
- **Present tense**: "Create an issue" not "Will create an issue"

### Content
- **Focus on what**: Describe action, not implementation
- **Omit obvious**: Don't explain basic concepts
- **Link for details**: Point to docs for deep explanations

## What NOT to Include

❌ **Getting Started sections** - No tutorials
❌ **Troubleshooting sections** - Minimal debugging info
❌ **Technical implementation details** - Save for docs
❌ **Extensive explanations** - Keep it terse
❌ **Multiple output examples** - Show command, not results
❌ **Configuration guides** - Reference only
❌ **Architecture explanations** - Link to docs

## Comparison: Before vs After

### Before (README-style)

```markdown
# shine start

Start or resume the shinectl service manager and all enabled panels.

## Description

The `start` command initializes the shine service by launching the
`shinectl` service manager. If shinectl is already running, the
command will report success without taking action.

When shinectl starts, it:
1. Reads the configuration from `~/.config/shine/prism.toml`
2. Spawns Kitty panels for each enabled prism via remote control
3. Launches `prismctl` supervisors to manage prism lifecycle
4. Creates IPC sockets in `/run/user/{uid}/shine/` for communication

## Examples

### Start the service for the first time

```bash
shine start
```

**Output**:
```
 ℹ Starting shinectl service...
✓ shinectl started (PID: 12345)
```

## Troubleshooting

### Service won't start

**Problem**: `shine start` fails with "shinectl not found in PATH"

**Solution**: Ensure shinectl is installed...
[15 more lines of troubleshooting]
```

### After (gh-style)

```markdown
# shine start

Start the shinectl service manager and all enabled panels.

If shinectl is already running, this command reports success without
taking action.

## USAGE
  shine start

## FLAGS
  --help   Show help for command

## BEHAVIOR
  When shinectl starts, it:
  - Reads configuration from ~/.config/shine/prism.toml
  - Spawns Kitty panels for each enabled prism
  - Launches prismctl supervisors
  - Creates IPC sockets in /run/user/{uid}/shine/

## EXAMPLES
  $ shine start
  $ shine start && shine status

## LEARN MORE
  Use `shine help status` to check running panels.
  Use `shine help logs` to view service logs.
  Config file: ~/.config/shine/prism.toml
```

**Reduction**: 139 lines → 28 lines (80% reduction)
**Scan time**: ~3 minutes → ~30 seconds (90% faster)

## Key Metrics

From analyzing `gh` help pages:

- **Opening line**: 8-15 words
- **Total length**:
  - Main help: ~60 lines
  - Group help: ~40 lines
  - Command help: ~30 lines
- **Examples**: 3-5 per command
- **Troubleshooting**: 0 lines (moved to docs)
- **Sections**: 5-7 per page

## Application to Shine

Applied to all shine help files:

- **usage.md**: 79 lines → 32 lines (59% reduction)
- **start.md**: 139 lines → 28 lines (80% reduction)
- **stop.md**: 161 lines → 23 lines (86% reduction)
- **status.md**: 229 lines → 28 lines (88% reduction)
- **reload.md**: 251 lines → 41 lines (84% reduction)
- **logs.md**: 270 lines → 35 lines (87% reduction)

**Total**: 1,129 lines → 187 lines (83% reduction)

## Reference

For canonical examples, run:
```bash
gh --help
gh repo --help
gh repo clone --help
gh issue create --help
```
