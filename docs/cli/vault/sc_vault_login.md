## sc vault login

Authenticate with a Vault server using username and password

### Synopsis

Authenticate with a HashiCorp Vault server using the "userpass" authentication method.

This command logs you into Vault using a username and password. If two-factor
authentication is enabled, you may also provide a one-time password (OTP).

After successful login, a Vault token is returned. The token is saved to a file.
If the file cannot be written—due to permission issues or missing directories—the token is printed
to stdout as a fallback.

```
sc vault login [flags]
```

### Options

```
  -h, --help              help for login
      --mfa-id string     MFA ID
  -m, --mount string      Vault mount for userpass auth engine (default "userpass")
  -o, --otp string        OTP value for non-interactive login
  -u, --username string   Username for login
```

### Options inherited from parent commands

```
      --no-telemetry     Do not perform check for updated version
      --profile string   Profile to use
  -v, --verbose          Print debug logs
```

### SEE ALSO

* [sc vault](sc_vault.md)	 - Commands for interacting with HashiCorp Vault

