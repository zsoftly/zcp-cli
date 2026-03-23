# Configuration Reference

This document describes all configuration options for ZCP CLI, including the config file structure, profile fields, environment variable overrides, and security considerations.

---

## Config File Location

ZCP CLI stores its configuration in a YAML file. The location depends on the operating system:

| Platform    | Default Path                        |
|-------------|-------------------------------------|
| Linux       | `~/.config/zcp/config.yaml`         |
| macOS       | `~/.config/zcp/config.yaml`         |
| Windows     | `%AppData%\zcp\config.yaml`         |

On Linux and macOS, the `XDG_CONFIG_HOME` environment variable is respected. If set, the config file will be located at `$XDG_CONFIG_HOME/zcp/config.yaml`.

The parent directory is created with `0700` permissions and the config file itself is written with `0600` permissions (owner read/write only) to prevent other users from reading credentials.

---

## Config File Structure

```yaml
active_profile: default

profiles:
  default:
    name: default
    apikey: YOUR_API_KEY
    secretkey: YOUR_SECRET_KEY
    api_url: ""              # Optional. Blank = use the default API URL.

  staging:
    name: staging
    apikey: STAGING_API_KEY
    secretkey: STAGING_SECRET_KEY
    api_url: https://staging.zcp.zsoftly.ca

  production:
    name: production
    apikey: PROD_API_KEY
    secretkey: PROD_SECRET_KEY
    api_url: ""
```

### Top-Level Fields

| Field            | Type   | Description                                              |
|------------------|--------|----------------------------------------------------------|
| `active_profile` | string | Name of the profile used when `--profile` is not set.   |
| `profiles`       | map    | Map of profile name to Profile object.                  |

---

## Profile Fields

Each profile supports the following fields:

| Field       | YAML Key    | Required | Description                                                                 |
|-------------|-------------|----------|-----------------------------------------------------------------------------|
| Name        | `name`      | Yes      | Must match the map key. Used for display in `zcp profile list`.            |
| API Key     | `apikey`    | Yes      | Your ZCP API key. Obtained from the ZSoftly Cloud Portal.                  |
| Secret Key  | `secretkey` | Yes      | Your ZCP secret key. Obtained from the ZSoftly Cloud Portal.               |
| API URL     | `api_url`   | No       | Override the API base URL for this profile. Blank uses the default.        |

The default API URL when `api_url` is blank or omitted is:

```
https://cloud.zcp.zsoftly.ca
```

---

## Environment Variable Overrides

The following environment variables are evaluated at runtime and take precedence over the corresponding config file values.

| Variable          | Overrides                       | Description                                                   |
|-------------------|---------------------------------|---------------------------------------------------------------|
| `ZCP_PROFILE`     | `--profile` flag / active profile | Profile name to use for the current invocation.             |
| `ZCP_API_KEY`     | Profile `apikey`                | API key, bypassing the config file entirely.                 |
| `ZCP_SECRET_KEY`  | Profile `secretkey`             | Secret key, bypassing the config file entirely.              |
| `ZCP_API_URL`     | Profile `api_url` / default URL | API base URL override.                                       |
| `XDG_CONFIG_HOME` | Config file directory (Linux/macOS) | Overrides the base directory for the config file.       |

Environment variables are useful for CI/CD pipelines where you do not want credentials stored in a file on disk.

Example usage in a pipeline:

```bash
export ZCP_API_KEY=ci-api-key
export ZCP_SECRET_KEY=ci-secret-key
zcp zone list --output json
```

---

## Multiple Profiles

You can configure multiple profiles for different environments (e.g., development, staging, production) and switch between them using `zcp profile use` or the `--profile` flag.

### Adding Profiles

```bash
# Interactive
zcp profile add

# Non-interactive
zcp profile add staging \
  --api-key YOUR_STAGING_KEY \
  --secret-key YOUR_STAGING_SECRET \
  --api-url https://staging.zcp.zsoftly.ca
```

### Switching the Active Profile

```bash
zcp profile use production
```

This updates `active_profile` in the config file.

### Per-Command Profile Override

```bash
zcp zone list --profile staging
```

The `--profile` flag does not modify the config file. It applies only to the current invocation.

### Listing Profiles

```bash
zcp profile list
```

Output includes the profile name, API URL, and whether it is currently active.

---

## Security Notes

**Never commit your config file to version control.** The file contains your API key and secret key in plaintext.

Add the config file to your `.gitignore`:

```
# .gitignore
.config/zcp/config.yaml
```

Additional recommendations:

- The config file is created with `0600` permissions. Do not change these permissions.
- If you suspect your credentials have been compromised, rotate your API key immediately in the ZSoftly Cloud Portal.
- In shared or CI environments, prefer environment variable injection (`ZCP_API_KEY`, `ZCP_SECRET_KEY`) over config files on disk.
- Do not log or print the config struct in scripts; the secret key will appear in output.
