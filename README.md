# WireGuard-API 🔐

WireGuard-API presents a JSON-RPC interface on top of a WireGuard network interface.

* 💖 **Add/Remove Peers**
  Modify known peers without reloading

* 📈 **Statistics**
  View data usage and allowed IPs of all peers

* 📞 **JSON-RPC 2.0 API**
  No custom client integrations required, standard API accepted everywhere.

**NOTE:** WireGuard-API is currently only compatible with the WireGuard Linux kernel module and userland wireguard-go. It does not currently work with the MacOS NetworkExtension.


## Getting WireGuard-API

### Pre-Built Binary

Binaries for Linux are available [here](https://github.com/jamescun/wireguard-api/releases).

### Build Yourself

WireGuard-API requires at least Go 1.13.

```sh
go install github.com/jamescun/wireguard-api/cmd
```

This should install the server binary `wireguard-api` in your $GOPATH/bin.

### Docker

WireGuard-API can also be run inside a Docker container, however the container will need to existing within the same network namespace as the host and have network administrator capability (CAP_NET_ADMIN) to be able to control the WireGuard interface.

```sh
docker run --name=wireguard-api -d -p 8080:8080 --network host --cap-add NET_ADMIN james/wireguard-api:latest wireguard-api --device=<my device>
```


## Configuring WireGuard-API

WireGuard is configured using command line arguments:

```sh
$ wireguard-api --help
WireGuard-API presents a JSON-RPC API to a WireGuard device
Usage: wireguard-api [options]

Helpers:
  --list-devices  list wireguard devices on this system and their name to be
                  given to --device
  --version       display the version number of WireGuard-API

Options:
  --device=<name>         (required) name of WireGuard device to manager
  --listen=<[host:]port>  address where API server will bind
                          (default localhost:8080)
  --tls                   enable Transport Layer Security (SSL) on server
  --tls-key               TLS private key
  --tks-cert              TLS certificate file
  --tls-client-ca         enable mutual TLS authentication (mTLS) of the client

Warnings:
  WireGuard-API can perform sensitive network operations, as such it should not
  be publically exposed. It should be bound to the local interface only, or
  failing that, be behind an authenticating proxy or have mTLS enabled.
```

The only required argument is `--device`, which tells WireGuard-API which WireGuard device to control. To control multiple WireGuard devices, launch multiple instances of WireGuard-API.

By default, this launches WireGuard-API on `localhost:8080` which may conflict with the typical development environment. To bind it elsewhere, use `--listen`:

```sh
$ wireguard-api --device=<my device> --listen=localhost:1234
```

**NOTE:** `--listen` will not prevent you from binding the server to a public interface. Care should be taken to prevent public access to the WireGuard-API server; such as binding it only to a local interface, placing an authenticating reverse proxy in-front of it or using mTLS (detailed below).

WireGuard-API can optional listen using TLS and HTTP/2. To enable TLS, you will also need a TLS Certificate and matching private key.

```sh
$ wireguard-api --device=<my device> --tls --tls-key=key.pem --tls-cert=cert.pem
```

And optionally WireGuard-API can request and validate client certificates to implement TLS Mutual Authentication (mTLS):

```sh
$ wireguard-api --device=<my device> --tls --tls-key=key.pem --tls-cert=cert.pem --tls-client-ca=clientca.pem
```


## Using WireGuard-API

WireGuard-API exposes a JSON-RPC 2.0 API with five methods.

All calls are made using the POST method, and require the `Content-Type` header to be set to `application/json`. The server ignores the URL path it is given, allowing the server to be mounted under another hierarchy in a reverse proxy.

The structures expected by the server can be found in [client/client.go](client/client.go).


### GetDeviceInfo

GetDeviceInfo returns information such as the public key and type of interface for the currently configured device.

```sh
curl http://localhost:8080 -H "Content-Type: application/json" -d '{"jsonrpc": "2.0", "method": "GetDeviceInfo", "params": {}}'
```

#### Example Response

```json
{
  "device": {
    "name": "wg0",
    "type": "Linux kernel",
    "public_key": "xoY2MZZ1UmbEakFBPyqryHwTaMi6ae4myP+vuILmJUY=",
    "listen_port": 51820,
    "num_peers": 13
  }
}
```


### ListPeers

ListPeers retrieves information about all Peers known to the current WireGuard interface, including allowed IP addresses and usage stats, optionally with pagination.

```sh
curl http://localhost:8080 -H "Content-Type: application/json" -d '{"jsonrpc": "2.0", "method": "ListPeers", "params": {}}'
```

#### Example Response

```json
{
  "peers": [
    {
      "public_key": "xoY2MZZ1UmbEakFBPyqryHwTaMi6ae4myP+vuILmJUY=",
      "has_preshared_key": false,
      "endpoint": "67.234.65.104:57436",
      "last_handshake": "2020-02-20T16:35:12Z",
      "receive_bytes": 834854756,
      "transmit_bytes": 3883746,
      "allowed_ips": [
        "10.1.1.0/24"
      ],
      "protocol_version": 1
    },
    ...
  ]
}
```


### GetPeer

GetPeer retrieves a specific Peer by their public key.

```sh
curl http://localhost:8080 -H "Content-Type: application/json" -d '{"jsonrpc": "2.0", "method": "GetPeer", "params": {"public_key": "xoY2MZZ1UmbEakFBPyqryHwTaMi6ae4myP+vuILmJUY="}}'
```

#### Example Response

```json
{
  "peer": {
    "public_key": "xoY2MZZ1UmbEakFBPyqryHwTaMi6ae4myP+vuILmJUY=",
    "has_preshared_key": false,
    "endpoint": "67.234.65.104:57436",
    "last_handshake": "2020-02-20T16:35:12Z",
    "receive_bytes": 834854756,
    "transmit_bytes": 3883746,
    "allowed_ips": [
      "10.1.1.0/24"
    ],
    "protocol_version": 1
  }
}
```


### AddPeer

AddPeer inserts a new Peer into the WireGuard interfaces table, multiple calls to AddPeer can be used to update details of the Peer.

```sh
curl http://localhost:8080 -H "Content-Type: application/json" -d '{"jsonrpc": "2.0", "method": "AddPeer", "params": {"public_key": "xoY2MZZ1UmbEakFBPyqryHwTaMi6ae4myP+vuILmJUY=","allowed_ips": [ "10.1.1.0/24" ]}}'
```


### RemovePeer

RemovePeer deletes a Peer from the WireGuard interfaces table by their public key,

```sh
curl http://localhost:8080 -H "Content-Type: application/json" -d '{"jsonrpc": "2.0", "method": "RemovePeer", "params": {"public_key": "xoY2MZZ1UmbEakFBPyqryHwTaMi6ae4myP+vuILmJUY="}}'
```
