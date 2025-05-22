# dotter 🚀

**dotter** is a command-line interface (CLI) tool, written in Go, for managing your dotfiles, shell configurations (aliases, functions), and ensuring your favorite shell tools are set up correctly. It's inspired by tools like Starship and aims to provide a declarative way to manage your shell environment through a simple TOML configuration file.

## Philosophy

- **Configuration over Code:** Define your environment declaratively in a `config.toml` file.
- **Idempotency:** Applying your configuration multiple times should result in the same state.
- **Simplicity:** Easy to understand and use for managing common dotfile and shell setups.
- **Extensibility (Future):** While the core is simple, the project considers future extensibility.

## Features (Planned & Implemented)

- Manage dotfiles via symlinking (with backup/overwrite/skip options).
- Process dotfiles with Go templating (access to environment variables and config values).
- Define and apply shell aliases and functions for various shells (Bash, Zsh, Fish).
- Inject necessary sourcing lines into your shell's RC file (`.bashrc`, `.zshrc`, `config.fish`).
- Check for the presence of specified tools and provide installation hints.
- `dotter init`: Initialize a new `dotter` configuration.
- `dotter apply`: Apply all defined configurations.
- `dotter list`: List managed items and their status.
- `dotter doctor`: Check the health of your `dotter` setup.
- `--dry-run` global flag: See what changes would be made without executing them.

## Installation

### Using `go install`

If you have Go installed and configured (Go version 1.18+ recommended):

```bash
go install github.com/mad01/dotter@latest
```

This will install the `dotter` binary to your Go bin directory (e.g., `$GOPATH/bin` or `$HOME/go/bin`). Ensure this directory is in your `PATH`.

### From Source

```bash
git clone https://github.com/mad01/dotter.git
cd dotter
go build -o dotter ./cmd/dotter/
# Then move `dotter` to a directory in your PATH, e.g., /usr/local/bin or ~/bin
```

### Binaries from Releases

(Once releases are set up) Pre-compiled binaries for various operating systems will be available on the [GitHub Releases](https://github.com/mad01/dotter/releases) page.

## Configuration (`config.toml`)

`dotter` uses a TOML file for configuration. By default, it looks for this file at:
- `$XDG_CONFIG_HOME/dotter/config.toml`
- `~/.config/dotter/config.toml` (if `$XDG_CONFIG_HOME` is not set)

**It is highly recommended to keep your `config.toml` within your version-controlled dotfiles repository and then symlink it to the default location.**

Run `dotter init` to create a default configuration file.

### Example `config.toml` Structure:

```toml
# Path to your dotfiles source repository (supports ~ expansion)
dotfiles_repo_path = "~/.dotfiles_src"

# --- Dotfiles Management --- 
# The key (e.g., "bashrc") is a logical name for the dotfile.
[dotfiles.bashrc]
source = ".bashrc"            # Relative path within dotfiles_repo_path
target = "~/.bashrc"          # Absolute path on the system (supports ~)
is_template = false           # Optional: set to true to process as Go template

[dotfiles.nvim_config]
source = "nvim/init.vim"
target = "~/.config/nvim/init.vim"

[dotfiles.gitconfig_template]
source = ".gitconfig.tmpl"
target = "~/.gitconfig"
is_template = true

# --- Tool Management --- 
# [[tools]]
# name = "fzf"
# check_command = "command -v fzf" # How dotter checks if the tool is installed
# install_hint = "Install fzf from https://github.com/junegunn/fzf"
# # Optional: manage config files for this tool using dotter
# config_files = [
#   { source = "fzf/.fzfrc", target = "~/.fzfrc" }
# ]

# --- Shell Configuration --- 
[shell.aliases]
ll = "ls -alhF"
g = "git"

[shell.functions.my_greeting]
body = '''
  echo "Hello from a dotter-managed function, $1!"
'''
```

### Templating

If `is_template = true` for a dotfile, it will be processed using Go's `text/template` engine before being symlinked. You can use:
- Environment variables: `{{ env "USER" }}`
- Configuration values: `{{ .DotterConfig.DotfilesRepoPath }}` (accesses the main `Config` struct)

## Usage

- `dotter init`: Guides you through creating an initial `config.toml`.
- `dotter apply`: Reads your `config.toml` and applies all configurations:
    - Symlinks dotfiles (processing templates if specified).
    - Generates shell alias and function scripts.
    - Injects sourcing lines into your shell's rc file.
- `dotter list`: Shows the status of managed dotfiles, configured tools, aliases, and functions.
- `dotter doctor`: Checks your setup for common issues (config validity, broken symlinks, rc file sourcing).
- `dotter --help`: Shows help for all commands and flags.

### Global Flags

- `-n`, `--dry-run`: Show what changes would be made without actually executing them.

## Best Practices

1.  **Version Control Your Dotfiles:** Keep your actual dotfiles (the source files) in a Git repository (e.g., `~/.dotfiles_src`).
2.  **Version Control `config.toml`:** Place your `dotter` `config.toml` file inside this same Git repository.
3.  **Symlink `config.toml`:** After placing `config.toml` in your dotfiles repository, symlink it to the expected location (`$XDG_CONFIG_HOME/dotter/config.toml` or `~/.config/dotter/config.toml`).
    Example: `ln -s ~/.dotfiles_src/config.toml ~/.config/dotter/config.toml`
4.  Run `dotter apply` whenever you make changes to your configuration or your dotfiles repository.

## Using `pkg/pipeutil` for Custom Shell Binaries

`dotter` includes a utility package `github.com/mad01/dotter/pkg/pipeutil` to help you write simple Go programs that can act as shell filters or transformers, easily interacting with stdin, stdout, and stderr.

### Features:

- `pipeutil.ReadAll()`: Reads all of `os.Stdin`.
- `pipeutil.Scanner()`: Returns a `bufio.Scanner` for line-by-line reading of `os.Stdin`.
- `pipeutil.Print([]byte)`: Writes to `os.Stdout`.
- `pipeutil.Println(string)`: Writes a string (with a newline) to `os.Stdout`.
- `pipeutil.Error(error)` / `pipeutil.Errorf(format, ...)`: Writes formatted errors to `os.Stderr`.
- `pipeutil.ExitSuccess` / `pipeutil.ExitFailure`: Constants for exit codes.

### Example (`pkg/pipeutil/example/main.go`):

```go
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mad01/dotter/pkg/pipeutil"
)

func main() {
	binput, err := pipeutil.ReadAll()
	if err != nil {
		pipeutil.Errorf("failed to read from stdin: %v", err)
		os.Exit(pipeutil.ExitFailure)
	}

	input := string(binput)
	upper := strings.ToUpper(input)

	_, err = pipeutil.Println(upper)
	if err != nil {
		pipeutil.Errorf("failed to write to stdout: %v", err)
		os.Exit(pipeutil.ExitFailure)
	}
	os.Exit(pipeutil.ExitSuccess)
}
```

**To compile and use such a binary:**

```bash
# Assuming you are in the directory of your custom tool (e.g., pkg/pipeutil/example)
go build -o mytool
echo "some input" | ./mytool
```

## Contributing

(CONTRIBUTING.md to be created if contributions are sought)

## License

(LICENSE file to be added - e.g., MIT, Apache 2.0)
