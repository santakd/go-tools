package server

import "net"

type THandle interface {
	Handle(conn *net.TCPConn)
}

// Wrap a panic, only print it, but ignore it.
//
// handle is a function, whose type is `func (*net.TCPConn)`, or a struct, which
// has implemented the interface `THandle`.
// wrap is the wrapper of *net.TCPConn, which sets the socket connection, and
// is used by TCPServerForever. In general, it is nil.
func TCPWrapError(conn *net.TCPConn, handle interface{}, wrap func(*net.TCPConn)) {
	yes := true
	defer func() {
		if err := recover(); err != nil {
			_logger.Printf("[Error] Get a error: %v", err)
			if !yes {
				panic(err)
			}
		}
	}()
	defer conn.Close()

	if wrap != nil {
		wrap(conn)
	}

	if handler, ok := handle.(THandle); ok {
		handler.Handle(conn)
	} else if handler, ok := handle.(func(*net.TCPConn)); ok {
		handler(conn)
	} else {
		yes = false
		panic("Don't support the handler")
	}
}

// Start a TCP server and never return. Return an error if returns.
//
// network MUST be "tcp", "tcp4", "tcp6".
// addr is like "host:port", such as "127.0.0.1:8000", and host or port
// may be omitted.
func TCPServerForever(network, addr string, handle interface{}, wrap func(*net.TCPConn)) error {
	var ln *net.TCPListener
	if _addr, err := net.ResolveTCPAddr(network, addr); err != nil {
		return err
	} else {
		if ln, err = net.ListenTCP(network, _addr); err != nil {
			return err
		}
	}

	defer ln.Close()

	_logger.Printf("[Debug] Listen on %v", addr)

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			_logger.Printf("[Error] Failed to AcceptTCP: %v", err)
		} else {
			go TCPWrapError(conn, handle, wrap)
		}
	}

	// Never execute forever.
	return nil
}