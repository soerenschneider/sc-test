## sc vault ssh

Manages the Vault SSH secret engine

```
sc vault ssh [flags]
```

### Options

```
  -h, --help           help for ssh
  -m, --mount string   Path where the SSH secret engine is mounted (default "ssh")
```

### Options inherited from parent commands

```
      --no-telemetry     Do not perform check for updated version
      --profile string   Profile to use
  -v, --verbose          Print debug logs
```

### SEE ALSO

* [sc vault](sc_vault.md)	 - Commands for interacting with HashiCorp Vault
* [sc vault ssh list-roles](sc_vault_ssh_list-roles.md)	 - Lists all roles for the SSH secrets engine
* [sc vault ssh read](sc_vault_ssh_read.md)	 - Reads ca data for the SSH secret engine
* [sc vault ssh sign-key](sc_vault_ssh_sign-key.md)	 - Signs a SSH public key

