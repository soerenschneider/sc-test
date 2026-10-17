## sc vault totp generate-secret

Manage Vault totp

### Synopsis

The 'token' command group contains subcommands for interacting with Vault tokens.

This command itself does not perform any actions. Instead, use one of its subcommands
to inspect or manage tokens.

```
sc vault totp generate-secret [flags]
```

### Options

```
  -e, --entity-id string   Identity Entity ID
  -f, --force              Force overwriting of existing TOTP secrets
  -h, --help               help for generate-secret
  -m, --method-id string   TOTP method ID
```

### Options inherited from parent commands

```
      --no-telemetry     Do not perform check for updated version
      --profile string   Profile to use
  -v, --verbose          Print debug logs
```

### SEE ALSO

* [sc vault totp](sc_vault_totp.md)	 - Manage Vault totp

