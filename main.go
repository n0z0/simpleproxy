package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

const (
	socks5Version = 0x05
	noAuth        = 0x00
	connectCmd    = 0x01
	udpAssociate  = 0x03
	ipv4Address   = 0x01
	domainName    = 0x03
	ipv6Address   = 0x04
)

var udpConn *net.UDPConn

func main() {
	port := "1080"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	// Start TCP listener
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to start TCP server: %v", err)
	}
	defer listener.Close()

	// Start UDP listener
	udpAddr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}
	udpConn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("Failed to start UDP server: %v", err)
	}
	defer udpConn.Close()

	log.Printf("SOCKS5 proxy server listening on port %s (TCP and UDP)", port)

	// Handle UDP relay in background
	go handleUDPRelay()

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

	// Handle different commands
	switch cmd {
	case connectCmd:
		// Continue with TCP CONNECT
	case udpAssociate:
		// Handle UDP ASSOCIATE
		return handleUDPAssociate(conn, addrType)
	default:
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
		0x00,       // Reserved
		0x01,       // IPv4
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

func handleUDPAssociate(conn net.Conn, addrType byte) (net.Conn, error) {
	// Read and discard the destination address and port from the request
	// For UDP ASSOCIATE, client sends desired address (usually 0.0.0.0:0)
	switch addrType {
	case ipv4Address:
		io.ReadFull(conn, make([]byte, 4))
	case domainName:
		lenBuf := make([]byte, 1)
		io.ReadFull(conn, lenBuf)
		io.ReadFull(conn, make([]byte, lenBuf[0]))
	case ipv6Address:
		io.ReadFull(conn, make([]byte, 16))
	}
	io.ReadFull(conn, make([]byte, 2)) // port

	// Get the UDP relay address
	udpAddr := udpConn.LocalAddr().(*net.UDPAddr)

	// Send reply with UDP relay address and port
	reply := []byte{
		socks5Version,
		0x00, // Success
		0x00, // Reserved
		0x01, // IPv4
	}
	// Add bind address (0.0.0.0 for simplicity)
	reply = append(reply, 0, 0, 0, 0)
	// Add bind port
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(udpAddr.Port))
	reply = append(reply, portBytes...)

	if _, err := conn.Write(reply); err != nil {
		return nil, fmt.Errorf("failed to send UDP ASSOCIATE reply: %w", err)
	}

	log.Printf("UDP ASSOCIATE established, relay port: %d", udpAddr.Port)

	// Keep the TCP connection alive for the UDP association
	// Return a dummy connection that will be closed when client disconnects
	return &dummyConn{conn}, nil
}

type dummyConn struct {
	net.Conn
}

func (d *dummyConn) Read(b []byte) (n int, err error) {
	// Block until client closes connection
	return d.Conn.Read(b)
}

func (d *dummyConn) Write(b []byte) (n int, err error) {
	return 0, nil // No-op
}

func handleUDPRelay() {
	buffer := make([]byte, 65535)
	for {
		n, clientAddr, err := udpConn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("UDP read error: %v", err)
			continue
		}

		go processUDPPacket(buffer[:n], clientAddr)
	}
}

func processUDPPacket(data []byte, clientAddr *net.UDPAddr) {
	// Parse SOCKS5 UDP request header
	if len(data) < 10 {
		log.Printf("UDP packet too short: %d bytes", len(data))
		return
	}

	// RSV (2 bytes) + FRAG (1 byte)
	if data[0] != 0 || data[1] != 0 {
		log.Printf("Invalid RSV field")
		return
	}

	frag := data[2]
	if frag != 0 {
		log.Printf("Fragmentation not supported")
		return
	}

	addrType := data[3]
	var targetAddr string
	var headerLen int

	switch addrType {
	case ipv4Address:
		if len(data) < 10 {
			return
		}
		targetAddr = net.IP(data[4:8]).String()
		headerLen = 10 // RSV(2) + FRAG(1) + ATYP(1) + ADDR(4) + PORT(2)

	case domainName:
		if len(data) < 5 {
			return
		}
		domainLen := int(data[4])
		if len(data) < 7+domainLen {
			return
		}
		targetAddr = string(data[5 : 5+domainLen])
		headerLen = 7 + domainLen

	case ipv6Address:
		if len(data) < 22 {
			return
		}
		targetAddr = net.IP(data[4:20]).String()
		headerLen = 22

	default:
		log.Printf("Unsupported address type: %d", addrType)
		return
	}

	// Extract port
	portOffset := headerLen - 2
	targetPort := binary.BigEndian.Uint16(data[portOffset : portOffset+2])

	// Extract payload
	payload := data[headerLen:]

	target := fmt.Sprintf("%s:%d", targetAddr, targetPort)

	// Forward to destination
	destAddr, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		log.Printf("Failed to resolve target %s: %v", target, err)
		return
	}

	// Create a temporary UDP connection for this request
	tempConn, err := net.DialUDP("udp", nil, destAddr)
	if err != nil {
		log.Printf("Failed to dial target %s: %v", target, err)
		return
	}
	defer tempConn.Close()

	// Send payload to target
	if _, err := tempConn.Write(payload); err != nil {
		log.Printf("Failed to send to target %s: %v", target, err)
		return
	}

	log.Printf("UDP relay: %s -> %s (%d bytes)", clientAddr, target, len(payload))

	// Wait for reply from target
	tempConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	replyBuf := make([]byte, 65535)
	n, err := tempConn.Read(replyBuf)
	if err != nil {
		// Timeout or error - this is normal for UDP
		return
	}

	// Build SOCKS5 UDP reply header
	reply := buildUDPReply(targetAddr, targetPort, addrType, replyBuf[:n])

	// Send reply back to client
	if _, err := udpConn.WriteToUDP(reply, clientAddr); err != nil {
		log.Printf("Failed to send reply to client: %v", err)
		return
	}

	log.Printf("UDP reply: %s <- %s (%d bytes)", clientAddr, target, n)
}

func buildUDPReply(targetAddr string, targetPort uint16, addrType byte, payload []byte) []byte {
	reply := []byte{0, 0, 0} // RSV + FRAG
	reply = append(reply, addrType)

	switch addrType {
	case ipv4Address:
		ip := net.ParseIP(targetAddr).To4()
		reply = append(reply, ip...)

	case domainName:
		reply = append(reply, byte(len(targetAddr)))
		reply = append(reply, []byte(targetAddr)...)

	case ipv6Address:
		ip := net.ParseIP(targetAddr).To16()
		reply = append(reply, ip...)
	}

	// Add port
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, targetPort)
	reply = append(reply, portBytes...)

	// Add payload
	reply = append(reply, payload...)

	return reply
}
