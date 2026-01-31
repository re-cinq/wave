---
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

## CLI Libraries and Frameworks

### 1. Go CLI Libraries
- **Cobra**: Powerful CLI framework for Go applications
- **urfave/cli**: Simple, fast, and fun CLI applications
- **flag**: Standard library flag package
- **pflag**: POSIX-compliant flag package
- **kingpin**: Deprioritized but still useful

### 2. Python CLI Libraries
- **Click**: Composable command interface creation
- **argparse**: Standard library argument parser
- **docopt**: Command-line interface descriptions
- **typer**: Modern CLI library with type hints
- **fire**: Automatic CLI generation

### 3. Node.js CLI Libraries
- **Commander.js**: Complete solution for Node.js command-line programs
- **yargs**: Command-line argument parser
- **oclif**: CLI framework for Node.js
- **meow**: Helper for CLI apps
- **minimist**: Argument parser

### 4. Rust CLI Libraries
- **clap**: Command Line Argument Parser
- **structopt**: Derive-based argument parser (deprecated, use clap)
- **argh**: Fast and simple argument parser
- **lexopt**: Minimalist argument parser

## Core CLI Concepts

### 1. Argument Parsing
- **Positional arguments**: Required arguments in specific positions
- **Optional flags**: Optional parameters with single/double dashes
- **Subcommands**: Nested command structures
- **Environment variables**: Configuration via environment
- **Config files**: Persistent configuration storage
- **Validation**: Type checking and value validation

### 2. Interactive Elements
- **Prompts**: User input with validation
- **Confirmations**: Yes/no confirmations
- **Selection menus**: Choose from predefined options
- **Progress bars**: Show operation progress
- **Spinners**: Indicate ongoing work

### 3. User Experience
- **Help systems**: Auto-generated help text
- **Error messages**: Clear, actionable error reporting
- **Auto-completion**: Tab completion for commands
- **Colors and formatting**: Readable output formatting
- **Consistency**: Follow CLI conventions

## CLI Development Patterns

### Go with Cobra Example
```go
package main

import (
    "fmt"
    "os"
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
    Use:   "myapp [command]",
    Short: "My application does awesome things",
    Long: `My application is a CLI tool that demonstrates 
best practices for command-line interface development.`,
}

var configCmd = &cobra.Command{
    Use:   "config [key] [value]",
    Short: "Get or set configuration values",
    Long:  `Get or set configuration values. If only key is provided,
gets the value. If both key and value are provided, sets the value.`,
    Args:  cobra.MinimumNArgs(1),
    Run:   runConfig,
}

var (
    configFile string
    verbose    bool
    output     string
)

func init() {
    cobra.OnInitialize(initConfig)
    
    rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file (default is $HOME/.myapp.yaml)")
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
    rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "json", "output format (json|yaml|text)")
    
    rootCmd.AddCommand(configCmd)
}

func initConfig() {
    if configFile != "" {
        viper.SetConfigFile(configFile)
    } else {
        home, err := os.UserHomeDir()
        if err != nil {
            fmt.Println(err)
            os.Exit(1)
        }
        
        viper.AddConfigPath(home)
        viper.SetConfigType("yaml")
        viper.SetConfigName(".myapp")
    }
    
    viper.AutomaticEnv()
    
    if err := viper.ReadInConfig(); err == nil {
        fmt.Println("Using config file:", viper.ConfigFileUsed())
    }
}

func runConfig(cmd *cobra.Command, args []string) {
    switch len(args) {
    case 1:
        // Get value
        value := viper.GetString(args[0])
        if value == "" {
            fmt.Printf("Config key '%s' not found\n", args[0])
            os.Exit(1)
        }
        fmt.Printf("%s: %s\n", args[0], value)
    case 2:
        // Set value
        viper.Set(args[0], args[1])
        fmt.Printf("Set %s = %s\n", args[0], args[1])
    default:
        fmt.Println("Usage: myapp config [key] [value]")
        os.Exit(1)
    }
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
```

### Python with Click Example
```python
#!/usr/bin/env python3
import click
import json
import sys
from pathlib import Path

@click.group()
@click.option('--config', '-c', type=click.Path(), help='Configuration file path')
@click.option('--verbose', '-v', is_flag=True, help='Enable verbose output')
@click.pass_context
def cli(ctx, config, verbose):
    """My application does awesome things."""
    ctx.ensure_object(dict)
    ctx.obj['config'] = config
    ctx.obj['verbose'] = verbose

@cli.command()
@click.argument('filename', type=click.Path(exists=True))
@click.option('--format', '-f', 
              type=click.Choice(['json', 'yaml', 'text']), 
              default='text', 
              help='Output format')
@click.pass_context
def process(ctx, filename, format):
    """Process a file and output results."""
    verbose = ctx.obj.get('verbose', False)
    
    if verbose:
        click.echo(f"Processing file: {filename}")
    
    try:
        with open(filename, 'r') as f:
            content = f.read()
        
        # Process the content
        result = process_content(content)
        
        # Output in requested format
        if format == 'json':
            click.echo(json.dumps(result, indent=2))
        elif format == 'yaml':
            import yaml
            click.echo(yaml.dump(result))
        else:
            click.echo(str(result))
            
    except Exception as e:
        click.echo(f"Error: {e}", err=True)
        sys.exit(1)

@cli.command()
@click.argument('key')
@click.argument('value', required=False)
@click.pass_context
def config(ctx, key, value):
    """Get or set configuration values."""
    config_file = ctx.obj.get('config') or get_default_config_path()
    
    if value:
        set_config_value(config_file, key, value)
        click.echo(f"Set {key} = {value}")
    else:
        value = get_config_value(config_file, key)
        if value:
            click.echo(f"{key} = {value}")
        else:
            click.echo(f"Config key '{key}' not found")
            sys.exit(1)

def process_content(content):
    """Example content processing function."""
    lines = content.split('\n')
    return {
        'lines': len(lines),
        'chars': len(content),
        'words': len(content.split())
    }

if __name__ == '__main__':
    cli()
```

### Rust with Clap Example
```rust
use clap::{Parser, Subcommand};
use serde::{Deserialize, Serialize};
use std::fs;
use std::io;

#[derive(Parser)]
#[command(author, version, about, long_about = None)]
struct Cli {
    #[arg(short, long, default_value = "config.yaml")]
    config: String,
    
    #[arg(short, long, action = clap::ArgAction::Count)]
    verbose: u8,
    
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    Process(ProcessCommand),
    Config(ConfigCommand),
}

#[derive(Parser)]
struct ProcessCommand {
    /// Input file to process
    #[arg(value_name = "FILE")]
    file: String,
    
    /// Output format
    #[arg(short, long, default_value = "text")]
    format: String,
}

#[derive(Parser)]
struct ConfigCommand {
    /// Configuration key to get/set
    key: String,
    
    /// Configuration value to set
    value: Option<String>,
}

fn main() {
    let cli = Cli::parse();
    
    match cli.command {
        Commands::Process(cmd) => process_file(cmd, &cli),
        Commands::Config(cmd) => handle_config(cmd, &cli),
    }
}

fn process_file(cmd: ProcessCommand, cli: &Cli) {
    if cli.verbose > 0 {
        println!("Processing file: {}", cmd.file);
    }
    
    match fs::read_to_string(&cmd.file) {
        Ok(content) => {
            let result = analyze_content(&content);
            
            match cmd.format.as_str() {
                "json" => println!("{}", serde_json::to_string_pretty(&result).unwrap()),
                "yaml" => println!("{}", serde_yaml::to_string(&result).unwrap()),
                _ => println!("{:?}", result),
            }
        }
        Err(e) => {
            eprintln!("Error reading file: {}", e);
            std::process::exit(1);
        }
    }
}

fn handle_config(cmd: ConfigCommand, cli: &Cli) {
    match cmd.value {
        Some(value) => set_config_value(&cli.config, &cmd.key, &value),
        None => {
            match get_config_value(&cli.config, &cmd.key) {
                Some(value) => println!("{} = {}", cmd.key, value),
                None => {
                    eprintln!("Config key '{}' not found", cmd.key);
                    std::process::exit(1);
                }
            }
        }
    }
}

#[derive(Serialize, Deserialize)]
struct ContentAnalysis {
    lines: usize,
    chars: usize,
    words: usize,
}

fn analyze_content(content: &str) -> ContentAnalysis {
    ContentAnalysis {
        lines: content.lines().count(),
        chars: content.chars().count(),
        words: content.split_whitespace().count(),
    }
}
```

## Interactive CLI Patterns

### Confirmation Prompts (Python with Click)
```python
import click

@click.command()
def deploy():
    """Deploy the application."""
    
    if not click.confirm('This will deploy to production. Continue?'):
        click.echo('Deployment cancelled.')
        return
    
    with click.progressbar(length=100, label='Deploying') as bar:
        for i in range(100):
            time.sleep(0.1)
            bar.update(1)
    
    click.echo('Deployment complete!')

@click.command()
@click.option('--force', is_flag=True, help='Skip confirmation')
def delete(force):
    """Delete resources."""
    
    if not force:
        if not click.confirm('This will delete all resources. Continue?'):
            click.echo('Deletion cancelled.')
            return
    
    # Perform deletion
    click.echo('Resources deleted.')
```

### Interactive Selection (Node.js with Inquirer)
```javascript
const inquirer = require('inquirer');
const program = require('commander');

program
    .version('1.0.0')
    .command('setup')
    .description('Interactive setup wizard')
    .action(async () => {
        const answers = await inquirer.prompt([
            {
                type: 'input',
                name: 'name',
                message: 'What is your project name?',
                validate: input => input.length > 0 || 'Project name is required'
            },
            {
                type: 'list',
                name: 'template',
                message: 'Choose a template:',
                choices: ['basic', 'advanced', 'minimal']
            },
            {
                type: 'checkbox',
                name: 'features',
                message: 'Select features:',
                choices: ['database', 'auth', 'logging', 'testing']
            }
        ]);
        
        console.log('Setup complete with:', answers);
        // Continue setup...
    });

program.parse(process.argv);
```

## CLI Testing Patterns

### Go CLI Testing
```go
package main

import (
    "bytes"
    "os"
    "strings"
    "testing"
    "github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        expected string
        error    bool
    }{
        {
            name:     "help flag",
            args:     []string{"--help"},
            expected: "myapp does awesome things",
            error:    false,
        },
        {
            name:     "invalid command",
            args:     []string{"invalid"},
            expected: "",
            error:    true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Capture output
            buf := new(bytes.Buffer)
            rootCmd.SetOut(buf)
            rootCmd.SetErr(buf)
            
            // Set arguments
            rootCmd.SetArgs(tt.args)
            
            // Execute command
            err := rootCmd.Execute()
            
            output := buf.String()
            
            if tt.error && err == nil {
                t.Errorf("expected error but got none")
            }
            if !tt.error && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
            if !strings.Contains(output, tt.expected) {
                t.Errorf("expected output to contain %q, got %q", tt.expected, output)
            }
        })
    }
}
```

### Python CLI Testing
```python
import pytest
from click.testing import CliRunner
from myapp import cli

def test_process_command(tmp_path):
    """Test the process command."""
    runner = CliRunner()
    
    # Create test file
    test_file = tmp_path / "test.txt"
    test_file.write_text("test content\n")
    
    # Run command
    result = runner.invoke(cli.process, [str(test_file)])
    
    assert result.exit_code == 0
    assert "lines: 1" in result.output
    assert "chars: 12" in result.output

def test_config_command():
    """Test the config command."""
    runner = CliRunner()
    
    # Test getting value
    result = runner.invoke(cli.config, ['test.key'])
    assert result.exit_code == 0
    
    # Test setting value
    result = runner.invoke(cli.config, ['test.key', 'test.value'])
    assert result.exit_code == 0
    assert "Set test.key = test.value" in result.output
```

## CLI Best Practices

### 1. Command Design
- Use descriptive command names (verbs are good)
- Follow Unix conventions (short options, long options)
- Provide help text and examples
- Support configuration files and environment variables
- Handle errors gracefully

### 2. Output Formatting
- Support multiple output formats (JSON, YAML, plain text)
- Use colors sparingly and respect NO_COLOR environment
- Provide progress indicators for long operations
- Format numbers with appropriate units

### 3. User Experience
- Implement auto-completion where possible
- Provide clear error messages with suggestions
- Use confirmation prompts for destructive operations
- Support verbose and quiet modes

### 4. Distribution
- Create single-binary executables where possible
- Provide installation instructions
- Include man pages or help documentation
- Consider packaging for different platforms

## When to Use This Skill

Use this skill when you need to:
- Build command-line tools and utilities
- Create CLI interfaces for existing applications
- Design interactive command-line applications
- Implement argument parsing and validation
- Add help systems and documentation to CLI tools
- Test CLI applications
- Package and distribute CLI tools

Always prioritize:
- Clear, intuitive command structures
- Comprehensive error handling
- Cross-platform compatibility
- Rich user experience when appropriate
- Comprehensive testing coverage