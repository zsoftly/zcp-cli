# Configuration Reference

This document describes all configuration options for ZCP CLI, including the config file structure, profile fields, environment variable overrides, and security considerations.

---

## Config File Location

ZCP CLI stores its configuration in a YAML file. The location depends on the operating system:

| Platform | Default Path                |
| -------- | --------------------------- |
| Linux    | `~/.config/zcp/config.yaml` |
| macOS    | `~/.config/zcp/config.yaml` |
| Windows  | `%AppData%\zcp\config.yaml` |

On Linux and macOS, the `XDG_CONFIG_HOME` environment variable is respected. If set, the config file will be located at `$XDG_CONFIG_HOME/zcp/config.yaml`.

The parent directory is created with `0700` permissions and the config file itself is written with `0600` permissions (owner read/write only) to prevent other users from reading credentials.

---

## Config File Structure

```yaml
active_profile: default

profiles:
  default:
    name: default
    bearer_token: YOUR_BEARER_TOKEN
    api_url: "" # Optional. Blank = use the default API URL.

  staging:
    name: staging
    bearer_token: STAGING_BEARER_TOKEN
    api_url: https://staging-api.zcp.zsoftly.ca

  production:
    name: production
    bearer_token: PROD_BEARER_TOKEN
    api_url: ""
```

### Top-Level Fields

| Field            | Type   | Description                                           |
| ---------------- | ------ | ----------------------------------------------------- |
| `active_profile` | string | Name of the profile used when `--profile` is not set. |
| `profiles`       | map    | Map of profile name to Profile object.                |

---

## Profile Fields

Each profile supports the following fields:

| Field          | YAML Key         | Required | Description                                                                                                                                                                 |
| -------------- | ---------------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Name           | `name`           | Yes      | Must match the map key. Used for display in `zcp profile list`.                                                                                                             |
| Bearer Token   | `bearer_token`   | Yes      | Your ZCP API bearer token. Obtained from the ZSoftly Cloud Portal.                                                                                                          |
| API URL        | `api_url`        | No       | Override the API base URL for this profile. Blank uses the default.                                                                                                         |
| Cloud Provider | `cloud_provider` | No       | Auto-detected and saved by `zcp auth validate` / `zcp profile add`; used by create commands so you need not pass `--cloud-provider`. Leave blank to let the CLI fill it in. |

The default API URL when `api_url` is blank or omitted is:

```
https://api.zcp.zsoftly.ca/api
```

---

## Environment Variable Overrides

The following environment variables are evaluated at runtime and take precedence over the corresponding config file values and global flags.

| Variable             | Overrides                           | Description                                                                                                                                         |
| -------------------- | ----------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ZCP_PROFILE`        | `--profile` flag / active profile   | Profile name to use for the current invocation.                                                                                                     |
| `ZCP_BEARER_TOKEN`   | Profile `bearer_token`              | Bearer token, bypassing the config file entirely.                                                                                                   |
| `ZCP_API_URL`        | Profile `api_url` / default URL     | API base URL override.                                                                                                                              |
| `ZCP_PROJECT`        | `--project` flag                    | Default project slug for all resource commands.                                                                                                     |
| `ZCP_REGION`         | `--region` flag                     | Default region slug for all resource commands.                                                                                                      |
| `ZCP_CLOUD_PROVIDER` | `--cloud-provider` flag / profile   | Cloud provider slug. Optional — auto-detected by `auth validate`; set only to override (multi-provider accounts, or CI that skips `auth validate`). |
| `ZCP_OUTPUT`         | `--output` / `-o` flag              | Default output format (`table`, `json`, `yaml`).                                                                                                    |
| `ZCP_DEBUG`          | `--debug` flag                      | Set to `true` to enable debug output (stderr).                                                                                                      |
| `XDG_CONFIG_HOME`    | Config file directory (Linux/macOS) | Overrides the base directory for the config file.                                                                                                   |

Environment variables are useful for CI/CD pipelines and scripting where you do not want to pass repetitive flags or store credentials in a file on disk.

> **Object storage uses its own provider and regions.** `object-storage` commands default to the `ceph` cloud provider (compute commands default to the auto-detected `nimbo`), so you do not normally set `ZCP_CLOUD_PROVIDER` for them. They also run in object-storage regions (`os-yul`, `os-yow`) rather than compute regions (`yul-1`, `yow-1`). Advanced bucket and object operations additionally connect **directly to the Ceph S3 (RGW) endpoint** read from the instance details — keep that endpoint reachable from any host (or CI runner) behind a proxy or firewall.

Example usage in a pipeline:

```bash
export ZCP_BEARER_TOKEN=ci-bearer-token
export ZCP_PROJECT=prod-project
export ZCP_REGION=yul-1
# Cloud provider is auto-detected by 'zcp auth validate'. In CI that skips it,
# set ZCP_CLOUD_PROVIDER to your provider slug (see 'zcp cloud-provider list').
export ZCP_OUTPUT=json

# Create a volume without passing repetitive flags
zcp volume create --name my-disk --plan b2g1 --billing-cycle hourly
```

### Local development: a sourced secrets file

For day-to-day terminal use, keep the token out of `.zshrc`/`.bashrc` and shell
history by placing the export in a locked-down file that your shell rc sources:

```bash
mkdir -p ~/.secrets && chmod 700 ~/.secrets
read -rs ZCP_TOKEN'?Paste ZCP token: '          # read from stdin — never lands in shell history
printf 'export ZCP_BEARER_TOKEN=%q\n' "$ZCP_TOKEN" > ~/.secrets/zcp.zsh && unset ZCP_TOKEN
chmod 600 ~/.secrets/zcp.zsh
echo '[ -f ~/.secrets/zcp.zsh ] && source ~/.secrets/zcp.zsh' >> ~/.zshrc
```

(Equivalently, create `~/.secrets/zcp.zsh` in your editor with the single line
`export ZCP_BEARER_TOKEN='<your-token>'`.) Avoid typing the token directly into
a command — anything on the command line is recorded in shell history.

The quoting matters: ZCP tokens contain a `|`, which an unquoted `export` line
treats as a shell pipe when sourced. `printf %q` (or single quotes) handles it.
Verify with an interactive subshell — `zsh -ic 'zcp auth validate'` — since
non-interactive `zsh -c` does not read `~/.zshrc`. When the env override is
active, `zcp auth validate` reports it explicitly
("Validating ZCP_BEARER_TOKEN (overrides profile …)"). To rotate, edit the
secrets file and open a new shell; to remove, delete the file (the rc line
checks existence first).

---

## Multiple Profiles

You can configure multiple profiles for different environments (e.g., development, staging, production) and switch between them using `zcp profile use` or the `--profile` flag.

### Adding Profiles

Every account starts with the project `default-9` (like `us-east-1` on AWS);
to use a different project, find its slug with `zcp project list`.

```bash
# Interactive — prompts for the bearer token
zcp profile add default --region yul-1 --project default-9

# Non-interactive
zcp profile add staging \
  --bearer-token YOUR_STAGING_TOKEN \
  --region yul-1 \
  --project default-9 \
  --api-url https://staging-api.zcp.zsoftly.ca
```

### Switching the Active Profile

```bash
zcp profile use production
```

This updates `active_profile` in the config file.

### Per-Command Profile Override

```bash
zcp region list --profile staging
```

The `--profile` flag does not modify the config file. It applies only to the current invocation.

### Listing Profiles

```bash
zcp profile list
```

Output includes the profile name, API URL, and whether it is currently active.

---

## Security Notes

**Never commit your config file to version control.** The file contains your bearer token in plaintext.

Add the config file to your `.gitignore`:

```
# .gitignore
.config/zcp/config.yaml
```

Additional recommendations:

- The config file is created with `0600` permissions. Do not change these permissions.
- If you suspect your credentials have been compromised, rotate your bearer token immediately in the ZSoftly Cloud Portal.
- In shared or CI environments, prefer environment variable injection (`ZCP_BEARER_TOKEN`) over config files on disk.
- Do not log or print the config struct in scripts; the bearer token will appear in output.
