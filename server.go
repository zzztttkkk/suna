package suna

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

type Logger interface {
	Print(v ...interface{})
	Printf(f string, v ...interface{})
	Println(v ...interface{})
}

type Server struct {
	Host      string
	Port      int
	TlsConfig *tls.Config

	Logger Logger

	BaseCtx context.Context

	Handler RequestHandler

	http1xProtocol Protocol
	http2Protocol  Protocol
}

type _CtxVKey int

const (
	CtxServerKey = _CtxVKey(iota)
	CtxConnKey
)

func (s Server) doListen() net.Listener {
	listener, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", s.Host, s.Port))
	if err != nil {
		panic(err)
	}
	return listener
}
func strSliceContains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func (s Server) enableTls(l net.Listener, certFile, keyFile string) net.Listener {
	if s.TlsConfig == nil {
		s.TlsConfig = &tls.Config{}
	}

	if !strSliceContains(s.TlsConfig.NextProtos, "http/1.1") {
		s.TlsConfig.NextProtos = append(s.TlsConfig.NextProtos, "http/1.1")
	}
	configHasCert := len(s.TlsConfig.Certificates) > 0 || s.TlsConfig.GetCertificate != nil
	if !configHasCert || certFile != "" || keyFile != "" {
		var err error
		s.TlsConfig.Certificates = make([]tls.Certificate, 1)
		s.TlsConfig.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			panic(err)
		}
	}
	return l
}

func (s Server) doAccept(l net.Listener) {
	var tempDelay time.Duration
	ctx := context.WithValue(s.BaseCtx, CtxServerKey, s)

	for {
		conn, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
		}

		go s.serve(context.WithValue(ctx, CtxConnKey, conn), conn)
	}
}

func (s *Server) ListenAndServe() {
	s.doAccept(s.doListen())
}

func (s *Server) ListenAndServeTLS(certFile, keyFile string) {
	s.doAccept(s.enableTls(s.doListen(), certFile, keyFile))
}

func (s *Server) tslHandshake(conn net.Conn) (*tls.Conn, string, error) {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return nil, "", nil
	}
	err := tlsConn.Handshake()
	if err != nil {
		return nil, "", err
	}
	return tlsConn, tlsConn.ConnectionState().NegotiatedProtocol, nil
}

func (s *Server) serve(ctx context.Context, conn net.Conn) {
	var protocolName string
	var tlsConn *tls.Conn
	var err error
	var protocol Protocol

	tlsConn, protocolName, err = s.tslHandshake(conn)
	if err != nil { // tls handshake error
		_ = conn.Close()
		return
	}

	if tlsConn == nil { // http1.x
		protocol = s.http1xProtocol
	} else {
		switch protocolName {
		case "", "http/1.0", "http/1.1":
			protocol = s.http1xProtocol
		case "http/2.0":
			protocol = s.http2Protocol
		}
	}

	if protocol == nil {
		_ = conn.Close()
		return
	}
	protocol.Serve(ctx, s, conn)
}
