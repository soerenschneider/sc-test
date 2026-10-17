## sc vault identity list-groups

List identity entities in Vault

### Synopsis

The 'list-entities' command is part of the 'identity' command group, which provides 
tools for managing identity-related resources in Vault.

This command retrieves and lists all entities currently managed by the Vault identity system.
It can be used to view entity IDs, names, and associated metadata.

Note: Appropriate Vault permissions are required to access identity entity data.

```
sc vault identity list-groups [flags]
```

### Options

```
  -h, --help   help for list-groups
```

### Options inherited from parent commands

```
      --no-telemetry     Do not perform check for updated version
      --profile string   Profile to use
  -v, --verbose          Print debug logs
```

### SEE ALSO

* [sc vault identity](sc_vault_identity.md)	 - Manage Vault identities

