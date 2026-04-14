---
name: cli
description: Expert command-line interface development including argument parsing, subcommands, interactive prompts, and CLI best practices
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

You are a Command Line Interface (CLI) expert specializing in argument parsing, subcommands, interactive prompts, and CLI best practices. Use this skill when the user needs help with:

- Creating command-line tools and utilities
- Implementing argument parsing and validation
- Building interactive CLI applications
- Designing CLI help systems and documentation
- CLI testing and distribution
- Cross-platform CLI development

## CLI Libraries (Quick Reference)

| Language | Primary Library | Notes |
|----------|----------------|-------|
| Go | Cobra + Viper | De facto standard for Go CLIs |
| Python | Click | Composable, decorator-based |
| Rust | clap | Derive-based, feature-rich |
| Node.js | Commander.js | Mature, widely used |

## Core CLI Concepts

### Argument Parsing
- **Positional arguments**: Required arguments in specific positions
- **Optional flags**: Parameters with `-s` / `--long` syntax
- **Subcommands**: Nested command structures (`app sub cmd`)
- **Environment variables**: `viper.AutomaticEnv()` / `click.option(envvar=...)`
- **Config files**: Persistent configuration layered below flags

### Interactive Elements
- Prompts, confirmations, selection menus, progress bars, spinners

## Key Patterns

### Go — Cobra + Viper (minimal skeleton)

```go
var rootCmd = &cobra.Command{Use: "myapp", Short: "Does awesome things"}
var verbose bool

func init() {
    cobra.OnInitialize(initConfig)
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
    rootCmd.PersistentFlags().StringP("output", "o", "json", "output format (json|yaml|text)")
    rootCmd.AddCommand(configCmd)
}

func initConfig() {
    viper.AddConfigPath(os.UserHomeDir())
    viper.SetConfigName(".myapp")
    viper.AutomaticEnv()
    viper.ReadInConfig()
}

func main() {
    if err := rootCmd.Execute(); err != nil { os.Exit(1) }
}
```

### Python — Click (group + command)

```python
@click.group()
@click.option('--verbose', '-v', is_flag=True)
@click.pass_context
def cli(ctx, verbose):
    ctx.ensure_object(dict)
    ctx.obj['verbose'] = verbose

@cli.command()
@click.argument('filename', type=click.Path(exists=True))
@click.option('--format', '-f', type=click.Choice(['json', 'yaml', 'text']), default='text')
@click.pass_context
def process(ctx, filename, format):
    if ctx.obj['verbose']:
        click.echo(f"Processing: {filename}")
    # ... process and output
```

### Rust — clap derive

```rust
#[derive(Parser)]
#[command(author, version, about)]
struct Cli {
    #[arg(short, long, default_value = "config.yaml")]
    config: String,
    #[arg(short, long, action = clap::ArgAction::Count)]
    verbose: u8,
    #[command(subcommand)]
    command: Commands,
}
```

### Interactive Prompts (Click)

```python
if not click.confirm('Deploy to production. Continue?'):
    click.echo('Cancelled.')
    return

with click.progressbar(items, label='Processing') as bar:
    for item in bar:
        process(item)
```

## Testing

```go
// Go: capture output, set args, execute
buf := new(bytes.Buffer)
rootCmd.SetOut(buf)
rootCmd.SetArgs([]string{"--help"})
err := rootCmd.Execute()
```

```python
# Python: Click test runner
runner = CliRunner()
result = runner.invoke(cli.process, [str(test_file)])
assert result.exit_code == 0
```

## Best Practices

1. **Command design**: Use verb-noun names, follow Unix conventions (`-s`/`--long`), always provide `--help`
2. **Output**: Support JSON/YAML/text; respect `NO_COLOR`; use progress indicators for long ops
3. **UX**: Confirm destructive ops; provide clear errors with suggestions; support `--verbose`/`--quiet`
4. **Distribution**: Single-binary where possible; provide shell completion scripts

## Complete Reference

For exhaustive patterns, examples, and advanced usage see:

**[`references/full-reference.md`](references/full-reference.md)**
