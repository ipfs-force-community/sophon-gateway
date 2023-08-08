package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gorilla/websocket"
)

func NewReverseServer(u *url.URL) http.Handler {
	urlForHttp := *u
	proxy := httputil.NewSingleHostReverseProxy(&urlForHttp)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "websocket" {
			proxy.ServeHTTP(w, r)
			return
		}

		// switch to websocket
		urlForWs := *r.URL
		switch u.Scheme {
		case "http":
			urlForWs.Scheme = "ws"
		case "https":
			urlForWs.Scheme = "wss"
		default:
			urlForWs.Scheme = "ws"
		}
		// todo: consider proxy on path
		urlForWs.Host = u.Host

		// clear up header
		header := http.Header{}
		for k, v := range r.Header {
			header[k] = v
		}
		for _, h := range []string{"Upgrade", "Connection", "Sec-Websocket-Key", "Sec-WebSocket-Version"} {
			header.Del(h)
		}

		proxyConn, resp, err := websocket.DefaultDialer.Dial(urlForWs.String(), header)
		if err != nil {
			err = fmt.Errorf("dial proxy websocket: %w", err)
			log.Error(err)
			log.Error(resp)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() {
			err := proxyConn.Close()
			if err != nil {
				log.Errorf("close proxyConn: %w", err)
			}
		}()

		upgrader := websocket.Upgrader{}
		clientConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			err = fmt.Errorf("upgrade websocket: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() {
			err := clientConn.Close()
			if err != nil {
				log.Errorf("close clientConn: %w", err)
			}
		}()

		signal := make(chan struct{})

		go forwardMessages(signal, proxyConn, clientConn)
		forwardMessages(signal, clientConn, proxyConn)
	})
}

func forwardMessages(signal chan struct{}, src *websocket.Conn, dst *websocket.Conn) {
	for {
		select {
		case <-signal:
			return
		default:
		}

		messageType, message, err := src.ReadMessage()
		if err != nil {
			log.Errorf("Failed to read message from %s: %w", src.RemoteAddr().String(), err)
			break
		}

		err = dst.WriteMessage(messageType, message)
		if err != nil {
			log.Errorf("Failed to write message to %s : %w", dst.RemoteAddr().String(), err)
			break
		}
	}

	signal <- struct{}{}
}
