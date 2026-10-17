## sc vault aws generate-credentials

Generate AWS credentials using Vault

### Synopsis

The 'generate-credentials' command is part of the 'aws' command group, which provides tools
for working with the Vault AWS secrets engine.

This command is used to generate dynamic AWS credentials by interacting with Vault. 
It requires appropriate configuration and permissions set in Vault to access the AWS secrets engine.

```
sc vault aws generate-credentials [flags]
```

### Options

```
  -p, --aws-profile string   Specifies the name of the AWS credentials profile section (default "default")
  -h, --help                 help for generate-credentials
  -m, --mount string         Mount path for the AWS secret engine (default "aws")
  -r, --role string          Specifies the name of the role to generate credentials for
  -t, --ttl int              Specify how long the credentials should be valid for in seconds (default 3600)
```

### Options inherited from parent commands

```
      --no-telemetry     Do not perform check for updated version
      --profile string   Profile to use
  -v, --verbose          Print debug logs
```

### SEE ALSO

* [sc vault aws](sc_vault_aws.md)	 - Manage AWS secret engine

