# Shell Completion

zcp supports shell completion for commands, subcommands, flags, and select arguments.

## Bash

Add to ~/.bashrc:

```
source <(zcp completion bash)
```

Or install system-wide:

```
zcp completion bash | sudo tee /etc/bash_completion.d/zcp
```

## Zsh

Add to ~/.zshrc:

```
source <(zcp completion zsh)
```

Or install to a fpath directory:

```
zcp completion zsh > "${fpath[1]}/_zcp"
```

## Fish

```
zcp completion fish | source
```

Or persist it:

```
zcp completion fish > ~/.config/fish/completions/zcp.fish
```

## What completes

- `zcp profile use <TAB>` — profile names from your config file
- `zcp profile delete <TAB>` — profile names from your config file
- `zcp profile show <TAB>` — profile names from your config file
- `--profile <TAB>` — profile names from your config file
- `--output <TAB>` — table, json, yaml
- All subcommands and flags complete automatically via Cobra
