## sc vault update-password

Update the password for Vault userpass authentication method

### Synopsis

Update the password on a HashiCorp Vault server using the "userpass" authentication method.

This command allows a user to change their Vault password by providing their.

```
sc vault update-password [flags]
```

### Options

```
  -h, --help              help for update-password
  -m, --mount string      Vault mount for userpass auth engine (default "userpass")
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

