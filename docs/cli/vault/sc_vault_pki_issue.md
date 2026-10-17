## sc vault pki issue

Issues a x509 certificate

```
sc vault pki issue [flags]
```

### Options

```
      --alt-names stringArray   Alternative names
      --ca-file string          File to save the ca to
  -c, --cert-file string        File to save the certificate to
  -n, --common-name string      The CN for the certificate
  -h, --help                    help for issue
  -s, --ip-sans stringArray     IP Sans
  -k, --key-file string         File to save the private key to
  -d, --min-duration string     Minimum duration of the cert before forcing issuing a new certificate (default "15m")
  -l, --min-lifetime float32    Minimum percentage of cert lifetime left before forcing issuing a new certificate (default 10)
  -r, --role string             Vault role
  -t, --ttl string              TTL of the certificate (default "24h")
```

### Options inherited from parent commands

```
  -m, --mount string     Path where the PKI secret engine is mounted (default "pki")
      --no-telemetry     Do not perform check for updated version
      --profile string   Profile to use
  -v, --verbose          Print debug logs
```

### SEE ALSO

* [sc vault pki](sc_vault_pki.md)	 - Manages the Vault PKI secret engine

