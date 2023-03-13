# Factory-cli: Booting from a live OS #

- [Factory-cli: Booting from a live OS](#factory-cli-booting-from-a-live-os)
  - [Background](#background)
  - [Boot from RHCOS live](#boot-from-rhcos-live)
    - [Example: Mounting on a Dell Server](#example-mounting-on-a-dell-server)
  - [Interacting with RHCOS live](#interacting-with-rhcos-live)
  - [Create a custom RHCOS live ISO](#create-a-custom-rhcos-live-iso)
    - [Verification](#verification)

## Background ##

We want to focus on target servers where only one disk is available and no external disk drive is possible to be
attached. Assisted installer, which is part of the ZTP flow, leverages the coreos-installer utility to write RHCOS to
disk. This means that if we boot from a pre-installed RHCOS on a single disk, the utility will complain because the
device is in use and cannot finish the process of writing. Then, the only way we have to run the full pre-caching
process is by booting from a live ISO and using the factory-cli tool from a container image to partition and pre-cache
all the artifacts required.

:warning: RHCOS requires the disk to not be in use when is about to be written by an RHCOS image. Reinstalling onto the
current boot disk is an unusual requirement, and the coreos-installer utility wasn't designed for it.

## Boot from RHCOS live ##

Technically you can boot from any live ISO that provides container tools such as podman. However, the supported and
tested OS is Red Hat CoreOS. You can obtain the latest live ISO from
[here](https://mirror.openshift.com/pub/openshift-v4/x86_64/dependencies/rhcos/latest/rhcos-live.x86_64.iso)

### Example: Mounting on a Dell Server ###

If you are running the install on a Dell Server you can use this simple script that leverages racadm to mount RHCOS live ISO from a local HTTPd server.

``` bash
#/bin/bash

CONTAINER_TOOL=podman
IDRAC_IP=$1

${CONTAINER_TOOL} run --network host --rm -it quay.io/alosadag/racadm:latest -r $IDRAC_IP -u root -p $PASSWD- remoteimage -d
${CONTAINER_TOOL} run --network host --rm -it quay.io/alosadag/racadm:latest -r $IDRAC_IP -u root -p $PASSWD- remoteimage -c -l http://10.19.138.94/rhcos-4.10.16-x86_64-live.x86_64.iso
${CONTAINER_TOOL} run --network host --rm -it quay.io/alosadag/racadm:latest -r $IDRAC_IP -u root -p $PASSWD- set iDRAC.VirtualMedia.BootOnce 1
${CONTAINER_TOOL} run --network host --rm -it quay.io/alosadag/racadm:latest -r $IDRAC_IP -u root -p $PASSWD- set iDRAC.ServerBoot.FirstBootDevice VCD-DVD
${CONTAINER_TOOL} run --network host --rm -it quay.io/alosadag/racadm:latest -r $IDRAC_IP -u root -p $PASSWD- serveraction powercycle
```

Then run the script:

``` console
./rhcos-live-racadm.sh 10.19.28.53
Security Alert: Certificate is invalid - self signed certificate
Continuing execution. Use -S option for racadm to stop execution on certificate-related errors.
Disable Remote File Started. Please check status using -s                    
option to know Remote File Share is ENABLED or DISABLED.

Security Alert: Certificate is invalid - self signed certificate
Continuing execution. Use -S option for racadm to stop execution on certificate-related errors.
Remote Image is now Configured                                               

Security Alert: Certificate is invalid - self signed certificate
Continuing execution. Use -S option for racadm to stop execution on certificate-related errors.
[Key=iDRAC.Embedded.1#VirtualMedia.1]                                        
Object value modified successfully

Security Alert: Certificate is invalid - self signed certificate
Continuing execution. Use -S option for racadm to stop execution on certificate-related errors.
[Key=iDRAC.Embedded.1#ServerBoot.1]                                          
Object value modified successfully

Security Alert: Certificate is invalid - self signed certificate
Continuing execution. Use -S option for racadm to stop execution on certificate-related errors.
Server power operation initiated successfully                        
```

>:warning: If you are targetting an HP server you can also use their own administration tools to manage HP iLOs.

You can also interact directly with the redfish interface, in case there is one available:

``` console
//Media Status
curl --globoff -H "Content-Type: application/json" -H "Accept: application/json" -k -X GET --user ${username_password} https://$BMC_ADDRESS/redfish/v1/Managers/Self/VirtualMedia/1 | python -m json.tool

//Insert Media. Please use your http server and iso file
curl --globoff -L -w "%{http_code} %{url_effective}\\n" -ku ${username_password} -H "Content-Type: application/json" -H "Accept: application/json" -d '{"Image": "http://[$HTTPd_IP]/RHCOS-live.iso"}' -X POST https://$BMC_ADDRESS/redfish/v1/Managers/Self/VirtualMedia/1/Actions/VirtualMedia.InsertMedia

// Set boot order
curl --globoff  -L -w "%{http_code} %{url_effective}\\n"  -ku ${username_password}  -H "Content-Type: application/json" -H "Accept: application/json" -d '{"Boot":{ "BootSourceOverrideEnabled": "Once", "BootSourceOverrideTarget": "Cd", "BootSourceOverrideMode": "UEFI"}}' -X PATCH https://$BMC_ADDRESS/redfish/v1/Systems/Self
```

Then reboot the server and make sure it is booting from virtual media.

![Booting from virtualmedia](images/idrac-virtualmedia.png "Booting from virtualmedia")

## Interacting with RHCOS live ##

Once you have mounted the live ISO using the IPMI interface and booted from virtual media you can connect to the virtual
console of your target server.  You should see that you are already logged into the system as the `core` user. In order
to make your life easier for setting up the next stage: [partitioning](../partitioning.md) we suggest permitting login
via SSH:

- Execute sudo to root
- Modify the /etc/ssh/sshd_config by allowing accessing using a password (PasswordAuthentication yes) and root as user (PermitRootLogin yes).
- Reload the sshd systemd service
- Change root password

At this point you should be able to SSH to the server and continue with the [partitioning](../partitioning.md) stage.

## Create a custom RHCOS live ISO ##

As we can see in the previous section, by default SSH server is not started in the live OS. The process mentioned above
is valid, but tedious. Also it is not suitable for automating the precache process. In such cases where there is a need
to automate the process we can leverage butane and the coreos-installer utilities.

Let's create a RHCOS live ISO with SSHd enabled and with some predefined credentials so it can be accessed right after booting.

First we need a download the RHCOS live ISO, for instance from
[here](https://mirror.openshift.com/pub/openshift-v4/x86_64/dependencies/rhcos/latest/rhcos-live.x86_64.iso). That live
ISO will be the base for creating a new one with the proper configuration.

Next, you need a yaml definition that will be used by [butane](https://coreos.github.io/butane/) to create an ignition file:

> :warning:  Remember when creating the password_hast that yescrypt passwords are not supported in RHCOS, use bcrypt instead.

```yaml
variant: rhcos
version: 0.1.0
passwd:
  users:
    - name: root
      ssh_authorized_keys:
        - 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCZnG8AIzlDAhpyENpK2qKiTT8EbRWOrz7NXjRzopbPu215mocaJgjjwJjh1cYhgPhpAp6M/ttTk7I4OI7g4588Apx4bwJep6oWTU35LkY8ZxkGVPAJL8kVlTdKQviDv3XX12l4QfnDom4tm4gVbRH0gNT1wzhnLP+LKYm2Ohr9D7p9NBnAdro6k++XWgkDeijLRUTwdEyWunIdW1f8G0Mg8Y1Xzr13BUo3+8aey7HLKJMDtobkz/C8ESYA/f7HJc5FxF0XbapWWovSSDJrr9OmlL9f4TfE+cQk3s+eoKiz2bgNPRgEEwihVbGsCN4grA+RzLCAOpec+2dTJrQvFqsD alosadag@sonnelicht.local'
      password_hash: '$2b$05$aAd9TkvWnvUmeDQEA9vkiOoCqCZgd6cWn0xXwe.ii/TqeCxUC0Z0G'
storage:
  files:
    - path: /etc/ssh/sshd_config
      contents:
        local: ./live-sshd_config
      mode: 0600
      overwrite: true
```

Notice that besides user and password, you can also include an authorized SSH key.

Next step is creating a folder called files-dir that will include the sshd_config that will overwrite the default sshd_config included in the live ISO.

An example of a valid sshd_config is shown here:

```console
$ cat files-dir/live-sshd_config 

#       $OpenBSD: sshd_config,v 1.103 2018/04/09 20:41:22 tj Exp $

# This is the sshd server system-wide configuration file.  See
# sshd_config(5) for more information.

# This sshd was compiled with PATH=/usr/local/bin:/usr/bin:/usr/local/sbin:/usr/sbin

# The strategy used for options in the default sshd_config shipped with
# OpenSSH is to specify options with their default value where
# possible, but leave them commented.  Uncommented options override the
# default value.

# If you want to change the port on a SELinux system, you have to tell
# SELinux about this change.
# semanage port -a -t ssh_port_t -p tcp #PORTNUMBER
#
#Port 22
#AddressFamily any
#ListenAddress 0.0.0.0
#ListenAddress ::

HostKey /etc/ssh/ssh_host_rsa_key
HostKey /etc/ssh/ssh_host_ecdsa_key
HostKey /etc/ssh/ssh_host_ed25519_key

# Ciphers and keying
#RekeyLimit default none

# This system is following system-wide crypto policy. The changes to
# crypto properties (Ciphers, MACs, ...) will not have any effect here.
# They will be overridden by command-line options passed to the server
# on command line.
# Please, check manual pages for update-crypto-policies(8) and sshd_config(5).

# Logging
#SyslogFacility AUTH
SyslogFacility AUTHPRIV
#LogLevel INFO

# Authentication:

#LoginGraceTime 2m
PermitRootLogin yes
#StrictModes yes
#MaxAuthTries 6
#MaxSessions 10

#PubkeyAuthentication yes

# The default is to check both .ssh/authorized_keys and .ssh/authorized_keys2
# but this is overridden so installations will only check .ssh/authorized_keys
AuthorizedKeysFile      .ssh/authorized_keys

#AuthorizedPrincipalsFile none

#AuthorizedKeysCommand none
#AuthorizedKeysCommandUser nobody

# For this to work you will also need host keys in /etc/ssh/ssh_known_hosts
#HostbasedAuthentication no
# Change to yes if you don't trust ~/.ssh/known_hosts for
# HostbasedAuthentication
#IgnoreUserKnownHosts no
# Don't read the user's ~/.rhosts and ~/.shosts files
#IgnoreRhosts yes

# To disable tunneled clear text passwords, change to no here!
#PasswordAuthentication yes
#PermitEmptyPasswords no
PasswordAuthentication yes

# Change to no to disable s/key passwords
#ChallengeResponseAuthentication yes
ChallengeResponseAuthentication no

# Kerberos options
#KerberosAuthentication no
#KerberosOrLocalPasswd yes
#KerberosTicketCleanup yes
#KerberosGetAFSToken no
#KerberosUseKuserok yes

# GSSAPI options
GSSAPIAuthentication yes
GSSAPICleanupCredentials no
#GSSAPIStrictAcceptorCheck yes
#GSSAPIKeyExchange no
#GSSAPIEnablek5users no

# Set this to 'yes' to enable PAM authentication, account processing,
# and session processing. If this is enabled, PAM authentication will
# be allowed through the ChallengeResponseAuthentication and
# PasswordAuthentication.  Depending on your PAM configuration,
# PAM authentication via ChallengeResponseAuthentication may bypass
# the setting of "PermitRootLogin without-password".
# If you just want the PAM account and session checks to run without
# PAM authentication, then enable this but set PasswordAuthentication
# and ChallengeResponseAuthentication to 'no'.
# WARNING: 'UsePAM no' is not supported in Fedora and may cause several
# problems.
UsePAM yes

#AllowAgentForwarding yes
#AllowTcpForwarding yes
#GatewayPorts no
X11Forwarding yes
#X11DisplayOffset 10
#X11UseLocalhost yes
#PermitTTY yes

# It is recommended to use pam_motd in /etc/pam.d/sshd instead of PrintMotd,
# as it is more configurable and versatile than the built-in version.
PrintMotd no

#PrintLastLog yes
#TCPKeepAlive yes
#PermitUserEnvironment no
#Compression delayed
ClientAliveInterval 180
#ClientAliveCountMax 3
#UseDNS no
#PidFile /var/run/sshd.pid
#MaxStartups 10:30:100
#PermitTunnel no
#ChrootDirectory none
#VersionAddendum none

# no default banner path
#Banner none

# Accept locale-related environment variables
AcceptEnv LANG LC_CTYPE LC_NUMERIC LC_TIME LC_COLLATE LC_MONETARY LC_MESSAGES
AcceptEnv LC_PAPER LC_NAME LC_ADDRESS LC_TELEPHONE LC_MEASUREMENT
AcceptEnv LC_IDENTIFICATION LC_ALL LANGUAGE
AcceptEnv XMODIFIERS

# override default of no subsystems
Subsystem       sftp    /usr/libexec/openssh/sftp-server

# Example of overriding settings on a per-user basis
#Match User anoncvs
#       X11Forwarding no
#       AllowTcpForwarding no
#       PermitTTY no
#       ForceCommand cvs server
```

Then, we are ready to run butane passing as arguments the previous yaml file and the folder we just created. Butane will transform the yaml custom configuration into an ignition file that coreos-installer can understand.

```console
butane -p embedded.yaml -d files-dir/ > embedded.ign
```

Once the ignition file is created, coreos-installer will help us to include the configuration into a new RHCOS live ISO named rhcos-sshd-4.11.0-x86_64-live.x86_64.iso:

```console
coreos-installer iso ignition embed -i embedded.ign rhcos-4.11.0-x86_64-live.x86_64.iso -o rhcos-sshd-4.11.0-x86_64-live.x86_64.iso
```

So, now we are ready to publish the new ISO (rhcos-sshd-4.11.0-x86_64-live.x86_64.iso) so it can be used to boot the server.

### Verification ###

It is suggested to verify that the new custom live ISO includes the configuration desired by running this command:

```console
coreos-installer iso ignition show rhcos-sshd-4.11.0-x86_64-live.x86_64.iso
```

```json

{
  "ignition": {
    "version": "3.2.0"
  },
  "passwd": {
    "users": [
      {
        "name": "root",
        "passwordHash": "$2b$05$aAd9TkvWnvUmeDQEA9vkiOoCqCZgd6cWn0xXwe.ii/TqeCxUC0Z0G",
        "sshAuthorizedKeys": [
          "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCZnG8AIzlDAhpyENpK2qKiTT8EbRWOrz7NXjRzopbPu215mocaJgjjwJjh1cYhgPhpAp6M/ttTk7I4OI7g4588Apx4bwJep6oWTU35LkY8ZxkGVPAJL8kVlTdKQviDv3XX12l4QfnDom4tm4gVbRH0gNT1wzhnLP+LKYm2Ohr9D7p9NBnAdro6k++XWgkDeijLRUTwdEyWunIdW1f8G0Mg8Y1Xzr13BUo3+8aey7HLKJMDtobkz/C8ESYA/f7HJc5FxF0XbapWWovSSDJrr9OmlL9f4TfE+cQk3s+eoKiz2bgNPRgEEwihVbGsCN4grA+RzLCAOpec+2dTJrQvFqsD alosadag@sonnelicht.local"
        ]
      }
    ]
  },
  "storage": {
    "files": [
      {
        "overwrite": true,
        "path": "/etc/ssh/sshd_config",
        "contents": {
          "compression": "gzip",
          "source": "data:;base64,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"
        },
        "mode": 384
      }
    ]
  }
}
```
