package client

import (
	"context"
	"time"
)

// Client is the interface expected to be presented to consumers of the API.
type Client interface {
	// GetDeviceInfo returns information such as the public key and type of
	// interface for the currently configured device.
	GetDeviceInfo(context.Context, *GetDeviceInfoRequest) (*GetDeviceInfoResponse, error)

	// ListPeers retrieves information about all Peers known to the current
	// WireGuard interface, including allowed IP addresses and usage stats,
	// optionally with pagination.
	ListPeers(context.Context, *ListPeersRequest) (*ListPeersResponse, error)

	// GetPeer retrieves a specific Peer by their public key.
	GetPeer(context.Context, *GetPeerRequest) (*GetPeerResponse, error)

	// AddPeer inserts a new Peer into the WireGuard interfaces table, multiple
	// calls to AddPeer can be used to update details of the Peer.
	AddPeer(context.Context, *AddPeerRequest) (*AddPeerResponse, error)

	// RemovePeer deletes a Peer from the WireGuard interfaces table by their
	// public key,
	RemovePeer(context.Context, *RemovePeerRequest) (*RemovePeerResponse, error)
}

type Device struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	PublicKey    string `json:"public_key"`
	ListenPort   int    `json:"listen_port"`
	FirewallMark int    `json:"firewall_mark,omitempty"`
	NumPeers     int    `json:"num_peers"`
}

type GetDeviceInfoRequest struct{}

type GetDeviceInfoResponse struct {
	Device *Device `json:"device"`
}

type Peer struct {
	PublicKey           string    `json:"public_key"`
	HasPresharedKey     bool      `json:"has_preshared_key"`
	Endpoint            string    `json:"endpoint"`
	PersistentKeepAlive string    `json:"persistent_keep_alive,omitempty"`
	LastHandshake       time.Time `json:"last_handshake"`
	ReceiveBytes        int64     `json:"receive_bytes"`
	TransmitBytes       int64     `json:"transmit_bytes"`
	AllowedIPs          []string  `json:"allowed_ips"`
	ProtocolVersion     int       `json:"protocol_version"`
}

type ListPeersRequest struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

type ListPeersResponse struct {
	Peers []*Peer `json:"peers"`
}

type GetPeerRequest struct {
	PublicKey string `json:"public_key"`
}

type GetPeerResponse struct {
	Peer *Peer `json:"peer"`
}

type AddPeerRequest struct {
	PublicKey           string   `json:"public_key"`
	PresharedKey        string   `json:"preshared_key,omitempty"`
	Endpoint            string   `json:"endpoint,omitempty"`
	PersistentKeepAlive string   `json:"persistent_keep_alive,omitempty"`
	AllowedIPs          []string `json:"allowed_ips,omitempty"`

	// ValidateOnly ensures only validation is completed, no side effects
	ValidateOnly bool `json:"validate_only"`
}

type AddPeerResponse struct {
	// OK will only ever be false if ValidateOnly has been requested.
	OK bool `json:"ok"`
}

type RemovePeerRequest struct {
	PublicKey string `json:"public_key"`

	// ValidateOnly ensures only validation is completed, no side effects
	ValidateOnly bool `json:"validate_only"`
}

type RemovePeerResponse struct {
	// OK will only ever be false if ValidateOnly has been requested.
	OK bool `json:"ok"`
}
