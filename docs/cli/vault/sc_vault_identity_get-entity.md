## sc vault identity get-entity

Retrieve a Vault identity entity

### Synopsis

The 'get-entity' command is part of the 'identity' command group, which provides 
functionality for interacting with Vault identity entities.

This command retrieves detailed information about a specific entity managed by the Vault identity system. 
It can be useful for inspecting entity metadata, aliases, and associated policies.

Note: This command may require appropriate Vault permissions to access identity resources.

```
sc vault identity get-entity [flags]
```

### Options

```
  -n, --entity-name string   Name of the entity
  -h, --help                 help for get-entity
```

### Options inherited from parent commands

```
      --no-telemetry     Do not perform check for updated version
      --profile string   Profile to use
  -v, --verbose          Print debug logs
```

### SEE ALSO

* [sc vault identity](sc_vault_identity.md)	 - Manage Vault identities

