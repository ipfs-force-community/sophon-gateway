package proxy

import (
	"fmt"
	"net/http"
	"net/url"

	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	maNet "github.com/multiformats/go-multiaddr/net"
)

var log = logging.Logger("proxy")

type IProxy interface {
	RegisterReverseHandler(hostKey HostKey, server http.Handler)
	RegisterReverseByAddr(hostKey HostKey, address string) error
	ProxyMiddleware(next http.Handler) http.Handler
}

// Proxy is a proxy for other component of venus chain service.
type Proxy struct {
	handler map[HostKey]http.Handler
	Key     map[string]HostKey
}

var _ IProxy = (*Proxy)(nil)

func NewProxy() *Proxy {
	p := &Proxy{
		handler: make(map[HostKey]http.Handler),
		Key:     make(map[string]HostKey),
	}
	for k, v := range Header2HostPreset {
		p.Key[k] = v
	}
	return p
}

func (p *Proxy) ProxyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiHeader := r.Header.Get(VenusAPINamespaceHeader)
		if apiHeader == "" {
			log.Debugf("no api header found, skip proxy")
			next.ServeHTTP(w, r)
			return
		}

		ser, err := p.getReverseHandler(apiHeader)
		if err != nil {
			log.Errorf("get reverse handler fail: %s", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ser.ServeHTTP(w, r)
	})
}

func (p *Proxy) getReverseHandler(header string) (http.Handler, error) {
	hostKey, ok := p.Key[header]
	if !ok {
		return nil, fmt.Errorf("header(%s): %w", header, ErrorInvalidHeader)
	}
	server, ok := p.handler[hostKey]
	if !ok {
		return nil, fmt.Errorf("host key(%s) : %w", hostKey, ErrorNoReverseProxyRegistered)
	}
	return server, nil
}

func (p *Proxy) RegisterReverseHandler(hostKey HostKey, server http.Handler) {
	if server == nil {
		delete(p.handler, hostKey)
		log.Info("unregister reverse proxy for ", hostKey)
		return
	}
	log.Infof("register reverse proxy for %s", hostKey)
	p.handler[hostKey] = server
}

func (p *Proxy) RegisterReverseByAddr(hostKey HostKey, address string) error {
	// unregister handler if address is empty
	if address == "" {
		delete(p.handler, hostKey)
		log.Info("unregister reverse proxy for ", hostKey)
		return nil
	}
	u, err := parseAddr(address)
	if err != nil {
		return err
	}

	log.Infof("register reverse proxy for %s: %s", hostKey, u.String())

	p.handler[hostKey] = NewReverseServer(u)
	return nil
}

// parseAddr parse a multiaddr or normal url string into url.Url
func parseAddr(address string) (*url.URL, error) {
	ma, err := multiaddr.NewMultiaddr(address)
	if err == nil {
		_, addr, err := maNet.DialArgs(ma)
		if err != nil {
			return nil, fmt.Errorf("parser libp2p url fail %w", err)
		}

		hasTLS := false

		_, err = ma.ValueForProtocol(multiaddr.P_WSS)
		if err == nil {
			hasTLS = true
		} else if err != multiaddr.ErrProtocolNotFound {
			return nil, err
		}

		_, err = ma.ValueForProtocol(multiaddr.P_HTTPS)
		if err == nil {
			hasTLS = true
		} else if err != multiaddr.ErrProtocolNotFound {
			return nil, err
		}

		if hasTLS {
			address = "https://" + addr
		} else {
			address = "http://" + addr
		}
	}

	return url.Parse(address)
}
