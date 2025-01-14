// Copyright 2019 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package net2

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

// TCPServerForever starts a TCP server. If starting successfully, never return.
func TCPServerForever(addr string, handler func(*net.TCPConn)) error {
	_addr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}

	ln, err := net.ListenTCP("tcp", _addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			fmt.Printf("AcceptTCP get an error: %v\n", err)
		} else {
			go handler(conn)
		}
	}

	// Never execute forever.
	// return nil
}

// TCPServer is used to manage a TCP server.
type TCPServer struct {
	Listener *net.TCPListener
	Handler  func(conn *net.TCPConn, isStopped func() bool)

	// When an error occurs, the error handler will be called.
	// If it returns true, the tcp server will continue to handle the connection.
	// Or, it will close the tcp server and return.
	//
	// The default error handler does nothing and returns false.
	ErrHandler func(error) bool

	once   sync.Once
	funcs  []func()
	conns  int64
	waits  sync.WaitGroup
	closed int32
}

// NewTCPServer returns a new TCPServer.
func NewTCPServer(ln *net.TCPListener, handler func(conn *net.TCPConn, isStopped func() bool)) *TCPServer {
	return &TCPServer{Listener: ln, Handler: handler}
}

// NewTCPServerFromAddr returns a new TCPServer listening on addr.
func NewTCPServerFromAddr(addr string, handler func(conn *net.TCPConn, isStopped func() bool)) (*TCPServer, error) {
	_addr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	ln, err := net.ListenTCP("tcp", _addr)
	if err != nil {
		return nil, err
	}

	return NewTCPServer(ln, handler), nil
}

// Start starts the TCP server.
func (s *TCPServer) Start() {
	errhandler := s.ErrHandler
	if errhandler == nil {
		errhandler = func(err error) bool { return false }
	}

	s.waits.Add(1)
	defer s.waits.Done()

	for {
		if s.IsStopped() {
			return
		}

		conn, err := s.Listener.AcceptTCP()
		if err != nil {
			if errhandler(err) {
				continue
			}
			s.once.Do(s.close)
			return
		}

		s.waits.Add(1)
		atomic.AddInt64(&s.conns, 1)
		go func() {
			defer func() {
				conn.Close()
				atomic.AddInt64(&s.conns, -1)
				s.waits.Done()
			}()

			s.Handler(conn, s.IsStopped)
		}()
	}
}

// Stop stops the TCP server.
func (s *TCPServer) Stop() {
	if atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		s.Listener.Close()
		s.once.Do(s.close)
	}
}

func (s *TCPServer) close() {
	for i := len(s.funcs) - 1; i >= 0; i-- {
		s.funcs[i]()
	}
}

// RegisterOnShutdown registers some callbacks, which will be called
// when the server is closed.
func (s *TCPServer) RegisterOnShutdown(callback ...func()) {
	s.funcs = append(s.funcs, callback...)
}

// Wait waits until all the connections are closed and exit.
func (s *TCPServer) Wait() {
	s.waits.Wait()
}

// IsStopped reports whether the TCP server is stopped
func (s *TCPServer) IsStopped() bool {
	return atomic.LoadInt32(&s.closed) == 1
}

// Connection reports the number of the client connection.
func (s *TCPServer) Connection() int {
	return int(atomic.LoadInt64(&s.conns))
}

// DialTCP dials a TCP connection to host:port.
func DialTCP(host, port interface{}) (*net.TCPConn, error) {
	return DialTCPByAddr(JoinHostPort(host, port))
}

// DialTCPByAddr dials a TCP connection to addr.
func DialTCPByAddr(addr string) (*net.TCPConn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return conn.(*net.TCPConn), nil
}
