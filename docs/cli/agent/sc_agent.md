## sc agent

Interact with a remote sc-agent instance

```
sc agent [flags]
```

### Options

```
      --ca-file string     The file that contains the x509 ca certificate
  -c, --cert-file string   The file that contains the x509 client certificate to authenticate with
  -h, --help               help for agent
  -k, --key-file string    The file that contains the x509 client key to authenticate with
      --server string      The endpoint of the server running sc-agent
  -v, --verbose            Increase verbosity of output
```

### Options inherited from parent commands

```
      --no-telemetry     Do not perform check for updated version
      --profile string   Profile to use
```

### SEE ALSO

* [sc](sc.md)	 - Universal Command Line Interface for soeren.cloud
* [sc agent certs](sc_agent_certs.md)	 - Interact with certificates
* [sc agent k0s](sc_agent_k0s.md)	 - Interact with k0s
* [sc agent libvirt](sc_agent_libvirt.md)	 - Interact with libvirt
* [sc agent login](sc_agent_login.md)	 - Get OIDC token
* [sc agent packages](sc_agent_packages.md)	 - Interact with the package component
* [sc agent power-state](sc_agent_power-state.md)	 - Interacts with the power-state component to either shutdown or reboot a machine.
* [sc agent replication](sc_agent_replication.md)	 - Interacts with the replication component
* [sc agent secrets](sc_agent_secrets.md)	 - Replicates secrets from Hashicorp Vault
* [sc agent service](sc_agent_service.md)	 - Interact with services component
* [sc agent wol](sc_agent_wol.md)	 - Interacts with the wake-on-lan component

