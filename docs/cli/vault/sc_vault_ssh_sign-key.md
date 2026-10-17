## sc vault ssh sign-key

Signs a SSH public key

```
sc vault ssh sign-key [flags]
```

### Options

```
  -c, --cert-file string         Where to save the certificate to
  -h, --help                     help for sign-key
      --principals stringArray   Principals
  -p, --pub-key-file string      Location of the public key to sign
  -r, --role string              Vault role
  -t, --ttl string               TTL of the certificate (default "24h")
```

### Options inherited from parent commands

```
  -m, --mount string     Path where the SSH secret engine is mounted (default "ssh")
      --no-telemetry     Do not perform check for updated version
      --profile string   Profile to use
  -v, --verbose          Print debug logs
```

### SEE ALSO

* [sc vault ssh](sc_vault_ssh.md)	 - Manages the Vault SSH secret engine

