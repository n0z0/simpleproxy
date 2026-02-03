package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

const (
	socks5Version = 0x05
	noAuth        = 0x00
	connectCmd    = 0x01
	ipv4Address   = 0x01
	domainName    = 0x03
	ipv6Address   = 0x04
)

func main() {
	port := "1080"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	log.Printf("SOCKS5 proxy server listening on port %s", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// SOCKS5 handshake
	if err := handshake(conn); err != nil {
		log.Printf("Handshake failed: %v", err)
		return
	}

	// Handle SOCKS5 request
	targetConn, err := handleRequest(conn)
	if err != nil {
		log.Printf("Request handling failed: %v", err)
		return
	}
	defer targetConn.Close()

	// Relay data between client and target
	relay(conn, targetConn)
}

func handshake(conn net.Conn) error {
	// Read version and number of methods
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return fmt.Errorf("failed to read handshake: %w", err)
	}

	version := buf[0]
	nMethods := buf[1]

	if version != socks5Version {
		return fmt.Errorf("unsupported SOCKS version: %d", version)
	}

	// Read methods
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return fmt.Errorf("failed to read methods: %w", err)
	}

	// Send response: version and selected method (no authentication)
	_, err := conn.Write([]byte{socks5Version, noAuth})
	return err
}

func handleRequest(conn net.Conn) (net.Conn, error) {
	// Read request header
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, fmt.Errorf("failed to read request: %w", err)
	}

	version := buf[0]
	cmd := buf[1]
	// buf[2] is reserved
	addrType := buf[3]

	if version != socks5Version {
		return nil, fmt.Errorf("unsupported SOCKS version: %d", version)
	}

	if cmd != connectCmd {
		sendReply(conn, 0x07) // Command not supported
		return nil, fmt.Errorf("unsupported command: %d", cmd)
	}

	// Parse target address
	var targetAddr string
	switch addrType {
	case ipv4Address:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return nil, fmt.Errorf("failed to read IPv4 address: %w", err)
		}
		targetAddr = net.IP(addr).String()

	case domainName:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return nil, fmt.Errorf("failed to read domain length: %w", err)
		}
		domain := make([]byte, lenBuf[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return nil, fmt.Errorf("failed to read domain: %w", err)
		}
		targetAddr = string(domain)

	case ipv6Address:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return nil, fmt.Errorf("failed to read IPv6 address: %w", err)
		}
		targetAddr = net.IP(addr).String()

	default:
		sendReply(conn, 0x08) // Address type not supported
		return nil, fmt.Errorf("unsupported address type: %d", addrType)
	}

	// Read port
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return nil, fmt.Errorf("failed to read port: %w", err)
	}
	port := binary.BigEndian.Uint16(portBuf)

	// Connect to target
	target := fmt.Sprintf("%s:%d", targetAddr, port)
	targetConn, err := net.Dial("tcp", target)
	if err != nil {
		sendReply(conn, 0x05) // Connection refused
		return nil, fmt.Errorf("failed to connect to target %s: %w", target, err)
	}

	// Send success reply
	if err := sendReply(conn, 0x00); err != nil {
		targetConn.Close()
		return nil, err
	}

	log.Printf("Connected to %s", target)
	return targetConn, nil
}

func sendReply(conn net.Conn, rep byte) error {
	// Reply format: VER REP RSV ATYP BND.ADDR BND.PORT
	reply := []byte{
		socks5Version,
		rep,
		0x00, // Reserved
		0x01, // IPv4
		0, 0, 0, 0, // Bind address (0.0.0.0)
		0, 0, // Bind port (0)
	}
	_, err := conn.Write(reply)
	return err
}

func relay(client, target net.Conn) {
	done := make(chan struct{}, 2)

	// Client -> Target
	go func() {
		io.Copy(target, client)
		done <- struct{}{}
	}()

	// Target -> Client
	go func() {
		io.Copy(client, target)
		done <- struct{}{}
	}()

	// Wait for one direction to finish
	<-done
}
