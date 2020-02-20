package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/jamescun/wg-api/client"
	"github.com/jamescun/wg-api/server/jsonrpc"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// Server is the host-side implementation of the WG-API Client. It supports
// both Kernel and Userland implementations of WireGuard.
type Server struct {
	wg         *wgctrl.Client
	deviceName string
}

// NewServer initializes a Server with a WireGuard client.
func NewServer(wg *wgctrl.Client, deviceName string) (*Server, error) {
	return &Server{wg: wg, deviceName: deviceName}, nil
}

// GetDeviceInfo returns information such as the public key and type of
// interface for the currently configured device.
func (s *Server) GetDeviceInfo(ctx context.Context, req *client.GetDeviceInfoRequest) (*client.GetDeviceInfoResponse, error) {
	dev, err := s.wg.Device(s.deviceName)
	if err != nil {
		return nil, fmt.Errorf("could not get WireGuard device: %w", err)
	}

	return &client.GetDeviceInfoResponse{
		Device: &client.Device{
			Name:         dev.Name,
			Type:         dev.Type.String(),
			PublicKey:    dev.PublicKey.String(),
			ListenPort:   dev.ListenPort,
			FirewallMark: dev.FirewallMark,
			NumPeers:     len(dev.Peers),
		},
	}, nil
}

func validateListPeersRequest(req *client.ListPeersRequest) error {
	if req == nil {
		return jsonrpc.InvalidParams("request body required", nil)
	}

	if req.Limit < 0 {
		return jsonrpc.InvalidParams("limit must be positive integer", nil)
	} else if req.Offset < 0 {
		return jsonrpc.InvalidParams("offset must be positive integer", nil)
	}

	return nil
}

// ListPeers retrieves information about all Peers known to the current
// WireGuard interface, including allowed IP addresses and usage stats,
// optionally with pagination.
func (s *Server) ListPeers(ctx context.Context, req *client.ListPeersRequest) (*client.ListPeersResponse, error) {
	if err := validateListPeersRequest(req); err != nil {
		return nil, err
	}

	dev, err := s.wg.Device(s.deviceName)
	if err != nil {
		return nil, fmt.Errorf("could not get WireGuard device: %w", err)
	}

	var peers []*client.Peer

	for _, peer := range dev.Peers {
		peers = append(peers, peer2rpc(peer))
	}

	// TODO(jc): pagination

	return &client.ListPeersResponse{
		Peers: peers,
	}, nil
}

func peer2rpc(peer wgtypes.Peer) *client.Peer {
	var keepAlive string
	if peer.PersistentKeepaliveInterval > 0 {
		keepAlive = peer.PersistentKeepaliveInterval.String()
	}

	var allowedIPs []string
	for _, allowedIP := range peer.AllowedIPs {
		allowedIPs = append(allowedIPs, allowedIP.String())
	}

	return &client.Peer{
		PublicKey:           peer.PublicKey.String(),
		HasPresharedKey:     peer.PresharedKey != wgtypes.Key{},
		Endpoint:            peer.Endpoint.String(),
		PersistentKeepAlive: keepAlive,
		LastHandshake:       peer.LastHandshakeTime,
		ReceiveBytes:        peer.ReceiveBytes,
		TransmitBytes:       peer.TransmitBytes,
		AllowedIPs:          allowedIPs,
		ProtocolVersion:     peer.ProtocolVersion,
	}
}

func validateGetPeerRequest(req *client.GetPeerRequest) error {
	if req == nil {
		return jsonrpc.InvalidParams("request body required", nil)
	}

	if req.PublicKey == "" {
		return jsonrpc.InvalidParams("public key is required", nil)
	} else if len(req.PublicKey) != 44 {
		return jsonrpc.InvalidParams("malformed public key", nil)
	}

	_, err := wgtypes.ParseKey(req.PublicKey)
	if err != nil {
		return jsonrpc.InvalidParams("invalid public key: "+err.Error(), nil)
	}

	return nil
}

// GetPeer retrieves a specific Peer by their public key.
func (s *Server) GetPeer(ctx context.Context, req *client.GetPeerRequest) (*client.GetPeerResponse, error) {
	if err := validateGetPeerRequest(req); err != nil {
		return nil, err
	}

	dev, err := s.wg.Device(s.deviceName)
	if err != nil {
		return nil, fmt.Errorf("could not get WireGuard device: %w", err)
	}

	publicKey, err := wgtypes.ParseKey(req.PublicKey)
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid public key: "+err.Error(), nil)
	}

	for _, peer := range dev.Peers {
		if peer.PublicKey == publicKey {
			return &client.GetPeerResponse{
				Peer: peer2rpc(peer),
			}, nil
		}
	}

	return &client.GetPeerResponse{}, nil
}

func validateAddPeerRequest(req *client.AddPeerRequest) error {
	if req == nil {
		return jsonrpc.InvalidParams("request body required", nil)
	}

	if req.PublicKey == "" {
		return jsonrpc.InvalidParams("public key is required", nil)
	} else if len(req.PublicKey) != 44 {
		return jsonrpc.InvalidParams("malformed public key", nil)
	}

	_, err := wgtypes.ParseKey(req.PublicKey)
	if err != nil {
		return jsonrpc.InvalidParams("invalid public key: "+err.Error(), nil)
	}

	if req.PresharedKey != "" {
		if len(req.PresharedKey) != 44 {
			return jsonrpc.InvalidParams("malformed preshared key", nil)
		}

		_, err := wgtypes.ParseKey(req.PresharedKey)
		if err != nil {
			return jsonrpc.InvalidParams("invalid preshared key: "+err.Error(), nil)
		}
	}

	if req.Endpoint != "" {
		_, err := net.ResolveUDPAddr("udp", req.Endpoint)
		if err != nil {
			return jsonrpc.InvalidParams("invalid endpoint: "+err.Error(), nil)
		}
	}

	if req.PersistentKeepAlive != "" {
		_, err := time.ParseDuration(req.PersistentKeepAlive)
		if err != nil {
			return jsonrpc.InvalidParams("invalid keepalive: "+err.Error(), nil)
		}
	}

	for _, allowedIP := range req.AllowedIPs {
		_, _, err := net.ParseCIDR(allowedIP)
		if err != nil {
			return jsonrpc.InvalidParams(fmt.Sprintf("range %q is not valid: %s", allowedIP, err), nil)
		}
	}

	return nil
}

// AddPeer inserts a new Peer into the WireGuard interfaces table, multiple
// calls to AddPeer can be used to update details of the Peer.
func (s *Server) AddPeer(ctx context.Context, req *client.AddPeerRequest) (*client.AddPeerResponse, error) {
	if err := validateAddPeerRequest(req); err != nil {
		return nil, err
	} else if req.ValidateOnly {
		return &client.AddPeerResponse{}, nil
	}

	publicKey, err := wgtypes.ParseKey(req.PublicKey)
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid public key: "+err.Error(), nil)
	}

	peer := wgtypes.PeerConfig{PublicKey: publicKey}

	if req.PresharedKey != "" {
		pk, err := wgtypes.ParseKey(req.PresharedKey)
		if err != nil {
			return nil, jsonrpc.InvalidParams("invalid preshared key: "+err.Error(), nil)
		}

		peer.PresharedKey = &pk
	}

	if req.Endpoint != "" {
		addr, err := net.ResolveUDPAddr("udp", req.Endpoint)
		if err != nil {
			return nil, jsonrpc.InvalidParams("invalid endpoint: "+err.Error(), nil)
		}

		peer.Endpoint = addr
	}

	if req.PersistentKeepAlive != "" {
		d, err := time.ParseDuration(req.PersistentKeepAlive)
		if err != nil {
			return nil, jsonrpc.InvalidParams("invalid keepalive: "+err.Error(), nil)
		}

		peer.PersistentKeepaliveInterval = &d
	}

	for _, allowedIP := range req.AllowedIPs {
		_, aip, err := net.ParseCIDR(allowedIP)
		if err != nil {
			return nil, jsonrpc.InvalidParams(fmt.Sprintf("range %q is not valid: %s", allowedIP, err), nil)
		}

		peer.AllowedIPs = append(peer.AllowedIPs, *aip)
	}

	err = s.wg.ConfigureDevice(s.deviceName, wgtypes.Config{Peers: []wgtypes.PeerConfig{peer}})
	if err != nil {
		return nil, fmt.Errorf("could not configure WireGuard device: %w", err)
	}

	return &client.AddPeerResponse{OK: true}, nil
}

func validateRemovePeerRequest(req *client.RemovePeerRequest) error {
	if req == nil {
		return jsonrpc.InvalidParams("request body required", nil)
	}

	if req.PublicKey == "" {
		return jsonrpc.InvalidParams("public key is required", nil)
	} else if len(req.PublicKey) != 44 {
		return jsonrpc.InvalidParams("malformed public key", nil)
	}

	_, err := wgtypes.ParseKey(req.PublicKey)
	if err != nil {
		return jsonrpc.InvalidParams("invalid public key: "+err.Error(), nil)
	}

	return nil
}

// RemovePeer deletes a Peer from the WireGuard interfaces table by their
// public key,
func (s *Server) RemovePeer(ctx context.Context, req *client.RemovePeerRequest) (*client.RemovePeerResponse, error) {
	if err := validateRemovePeerRequest(req); err != nil {
		return nil, err
	} else if req.ValidateOnly {
		return &client.RemovePeerResponse{}, nil
	}

	publicKey, err := wgtypes.ParseKey(req.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	peer := wgtypes.PeerConfig{
		PublicKey: publicKey,
		Remove:    true,
	}

	err = s.wg.ConfigureDevice(s.deviceName, wgtypes.Config{Peers: []wgtypes.PeerConfig{peer}})
	if err != nil {
		return nil, fmt.Errorf("could not configure WireGuard device: %w", err)
	}

	return &client.RemovePeerResponse{OK: true}, nil
}

// ServeJSONRPC handles incoming WG-API requests.
func (s *Server) ServeJSONRPC(w jsonrpc.ResponseWriter, r *jsonrpc.Request) {
	var res interface{}

	// TODO(jc): must be a way to make this generic, reflection maybe?

	switch r.Method {
	case "GetDeviceInfo":
		var err error
		res, err = s.GetDeviceInfo(r.Context(), &client.GetDeviceInfoRequest{})
		if err != nil {
			res = jsonrpc.ServerError(-32000, err.Error(), nil)
		}

	case "ListPeers":
		var arg client.ListPeersRequest
		err := json.Unmarshal(r.Params, &arg)
		if err != nil {
			res = jsonrpc.ParseError(err.Error(), nil)
		} else {
			res, err = s.ListPeers(r.Context(), &arg)
			if err != nil {
				res = jsonrpc.ServerError(-32000, err.Error(), nil)
			}
		}

	case "GetPeer":
		var arg client.GetPeerRequest
		err := json.Unmarshal(r.Params, &arg)
		if err != nil {
			res = jsonrpc.ParseError(err.Error(), nil)
		} else {
			res, err = s.GetPeer(r.Context(), &arg)
			if err != nil {
				res = jsonrpc.ServerError(-32000, err.Error(), nil)
			}
		}

	case "AddPeer":
		var arg client.AddPeerRequest
		err := json.Unmarshal(r.Params, &arg)
		if err != nil {
			res = jsonrpc.ParseError(err.Error(), nil)
		} else {
			res, err = s.AddPeer(r.Context(), &arg)
			if err != nil {
				res = jsonrpc.ServerError(-32000, err.Error(), nil)
			}
		}

	case "RemovePeer":
		var arg client.RemovePeerRequest
		err := json.Unmarshal(r.Params, &arg)
		if err != nil {
			res = jsonrpc.ParseError(err.Error(), nil)
		} else {
			res, err = s.RemovePeer(r.Context(), &arg)
			if err != nil {
				res = jsonrpc.ServerError(-32000, err.Error(), nil)
			}
		}

	default:
		res = jsonrpc.MethodNotFound("method not found", nil)
	}

	w.Write(res)
}
