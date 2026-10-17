## sc agent login

Get OIDC token

```
sc agent login [flags]
```

### Options

```
  -i, --client-id string   OIDC client id (default "sc_agent")
  -h, --help               help for login
  -p, --provider string    Username for login (default "https://auth.dd.soeren.cloud/realms/soerencloud")
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

* [sc agent](sc_agent.md)	 - Interact with a remote sc-agent instance

