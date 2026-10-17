## sc agent power-state

Interacts with the power-state component to either shutdown or reboot a machine.

```
sc agent power-state [flags]
```

### Options

```
  -h, --help   help for power-state
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
* [sc agent power-state reboot](sc_agent_power-state_reboot.md)	 - Reboots the machine
* [sc agent power-state shutdown](sc_agent_power-state_shutdown.md)	 - Shuts a machine down

