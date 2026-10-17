## sc vault aws list

Lists all existing roles in the AWS secrets engine.

### Synopsis

List all configured roles in the Vault AWS secrets engine.

This command queries the AWS secrets engine mounted in Vault to retrieve a list of
all available role names. These roles define the IAM permissions that Vault can
generate for temporary AWS credentials.

The command expects that the AWS secrets engine has been enabled and configured.

```
sc vault aws list [flags]
```

### Options

```
  -h, --help           help for list
  -m, --mount string   Mount path for the AWS secret engine (default "aws")
```

### Options inherited from parent commands

```
      --no-telemetry     Do not perform check for updated version
      --profile string   Profile to use
  -v, --verbose          Print debug logs
```

### SEE ALSO

* [sc vault aws](sc_vault_aws.md)	 - Manage AWS secret engine

