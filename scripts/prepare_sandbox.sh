#!/bin/bash
set -e

# Create the sandbox directory
mkdir -p /sandbox

# Create a sample dotfiles repository with some example files
mkdir -p /sandbox/dotfiles_repo
echo 'export PS1="\033[1;32m[dotter-sandbox]\033[0m \w $ "' > /sandbox/dotfiles_repo/.bashrc
echo 'set -g default-terminal "screen-256color"' > /sandbox/dotfiles_repo/.tmux.conf
echo 'syntax on
set number
set tabstop=4
colorscheme desert' > /sandbox/dotfiles_repo/.vimrc
echo 'alias ll="ls -alh"' > /sandbox/dotfiles_repo/.aliases
chmod -R 755 /sandbox

# Create pre/post hook scripts
mkdir -p /sandbox/dotfiles_repo/scripts
echo '#!/bin/sh
echo "Running pre-apply hook script! This would back up your dotfiles."' > /sandbox/dotfiles_repo/scripts/pre-apply.sh
echo '#!/bin/sh
echo "Running post-link hook for .bashrc!"' > /sandbox/dotfiles_repo/scripts/post-bashrc.sh
chmod +x /sandbox/dotfiles_repo/scripts/*.sh

# Create the setup script
cat > /sandbox/setup.sh << 'EOF'
#!/bin/bash
mkdir -p ~/.config/dotter

# Create config.toml directly
cat > ~/.config/dotter/config.toml << EOFCONFIG
dotfiles_repo_path = "/sandbox/dotfiles_repo"

[dotfiles.bashrc]
source = ".bashrc"
target = "~/.bashrc"

[dotfiles.vimrc]
source = ".vimrc"
target = "~/.vimrc"

[dotfiles.tmux]
source = ".tmux.conf"
target = "~/.tmux.conf"

[shell.aliases]
ll = "ls -alh"
la = "ls -A"
l = "ls -CF"

[shell.functions.dotter_help]
body = """
echo "Welcome to the dotter sandbox!"
echo "Try these commands:"
echo "  dotter apply - Apply your dotfile configurations"
echo "  dotter list - List managed dotfiles and their status"
echo "  dotter doctor - Check the health of your dotter setup"
echo "  vim ~/.config/dotter/config.toml - Edit your dotter config"
"""

# Hook configuration
[hooks]
# Hooks to run before any dotfiles are processed
pre_apply = [
  "echo \"Starting dotfile application process\"",
  "/sandbox/dotfiles_repo/scripts/pre-apply.sh",
]

# Hooks to run after all dotfiles are processed
post_apply = [
  "echo \"All dotfiles have been processed\"",
]

# Hooks to run after specific dotfiles are linked
[hooks.post_link]
# After linking .bashrc
bashrc = [
  "/sandbox/dotfiles_repo/scripts/post-bashrc.sh",
]
EOFCONFIG

echo "Welcome to the dotter sandbox!"
echo "Your dotfiles repo is at: /sandbox/dotfiles_repo"
echo "Your dotter config is at: ~/.config/dotter/config.toml"
echo ""
echo "This sandbox includes the new HOOKS SYSTEM for extending dotter functionality."
echo ""
echo "Try these commands:"
echo "  dotter apply   - Apply your dotfile configurations and see hooks in action"
echo "  dotter list    - List managed dotfiles and their status"
echo "  dotter doctor  - Check the health of your dotter setup"
echo "  cd /sandbox    - Explore the sandbox environment"
echo "  vim ~/.config/dotter/config.toml - Examine the hooks configuration"
echo ""
echo "Changes within this container will be lost when you exit (due to --rm)."
exec bash
EOF

chmod +x /sandbox/setup.sh 