## sc vault totp generate-method

Manage Vault totp

### Synopsis

The 'token' command group contains subcommands for interacting with Vault tokens.

This command itself does not perform any actions. Instead, use one of its subcommands
to inspect or manage tokens.

```
sc vault totp generate-method [flags]
```

### Options

```
      --algorithm string     Specifies the hashing algorithm used to generate the TOTP code. Options include "SHA1", "SHA256" and "SHA512" (default "SHA256")
  -h, --help                 help for generate-method
  -i, --issuer string        The name of the key's issuing organization
  -m, --method-name string   The unique name identifier for this MFA method
```

### Options inherited from parent commands

```
      --no-telemetry     Do not perform check for updated version
      --profile string   Profile to use
  -v, --verbose          Print debug logs
```

### SEE ALSO

* [sc vault totp](sc_vault_totp.md)	 - Manage Vault totp

