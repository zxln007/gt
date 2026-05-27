//go:build !cgo
// +build !cgo

package msquic

import (
	"crypto/tls"
	"errors"
	"net"
)

func MsquicDial(addr string, config *tls.Config) (conn net.Conn, err error) {
	return nil, errors.New("msquic is not supported without cgo")
}

func MsquicListen(addr string, keyFile string, certFile string) (net.Listener, error) {
	return nil, errors.New("msquic is not supported without cgo")
}
