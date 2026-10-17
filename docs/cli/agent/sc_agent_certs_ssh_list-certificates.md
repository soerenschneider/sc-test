## sc agent certs ssh list-certificates

List all managed SSH certificates

```
sc agent certs ssh list-certificates [flags]
```

### Options

```
      --cert-type string   The type of certs to return, must be either 'user' or 'host' (default "user")
  -h, --help               help for list-certificates
```

### Options inherited from parent commands

```
      --ca-file string     The file that contains the x509 ca certificate
  -c, --cert-file string   The file that contains the x509 client certificate to authenticate with
  -k, --key-file string    The file that contains the x509 client key to authenticate with
      --no-telemetry       Do not perform check for updated version
      --profile string     Profile to use
      --server string      The endpoint of the server running sc-agent
  -v, --verbose            Increase verbosity of output
```

### SEE ALSO

* [sc agent certs ssh](sc_agent_certs_ssh.md)	 - Manage ssh certificates

