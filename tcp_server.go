package tcp_server

import (
	"bufio"
	"crypto/tls"
	"log"
	"net"
	"io"
	"encoding/binary"
	"bytes"
)

// Client holds info about connection
type Client struct {
	conn   net.Conn
	Server *server
}

// TCP server
type server struct {
	address                  string // Address to open connection: localhost:9999
	config                   *tls.Config
	onNewClientCallback      func(c *Client)
	onClientConnectionClosed func(c *Client, err error)
	onNewMessage             func(c *Client, message []byte)
}

// Read client data from channel
func (c *Client) listen(packge_type string) {
	c.Server.onNewClientCallback(c)
	reader := bufio.NewReader(c.conn)
	if packge_type == "by_str" {
		for {
			message, err := reader.ReadBytes('\n')
			if err != nil {
				c.conn.Close()
				c.Server.onClientConnectionClosed(c, err)
				return
			}
			c.Server.onNewMessage(c, message)
		}
	}else if packge_type == "lv" {
		for {
			//1 先读出流中的head部分
			headData := make([]byte, 4)
			_, err := io.ReadFull(c.conn, headData) //ReadFull 会把msg填充满为止
			if err != nil {
				log.Fatalln("read head error")
				break
			}

			
			var len int32
			buf := bytes.NewBuffer(headData)
			err = binary.Read(buf, binary.BigEndian, &len)
			if err != nil {
				log.Fatalln("binary.Read failed:", err)
				break
			}
			log.Printf("recv head , datalen:%d", len)

			body_data := make([]byte, len)
			n := 0
			n, err = io.ReadFull(c.conn, body_data) //ReadFull 会把msg填充满为止
			if err != nil {
				log.Fatalln("read body error")
				break
			}
			log.Printf("recv data, len:%d", n)

			c.Server.onNewMessage(c, body_data)
		}
	}
}

// Send text message to client
func (c *Client) Send(message string) error {
	return c.SendBytes([]byte(message))
}

// Send bytes to client
func (c *Client) SendBytes(b []byte) error {
	_, err := c.conn.Write(b)
	if err != nil {
		c.conn.Close()
		c.Server.onClientConnectionClosed(c, err)
	}
	return err
}

func (c *Client) Conn() net.Conn {
	return c.conn
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// Called right after server starts listening new client
func (s *server) OnNewClient(callback func(c *Client)) {
	s.onNewClientCallback = callback
}

// Called right after connection closed
func (s *server) OnClientConnectionClosed(callback func(c *Client, err error)) {
	s.onClientConnectionClosed = callback
}

// Called when Client receives new message
func (s *server) OnNewMessage(callback func(c *Client, message []byte)) {
	s.onNewMessage = callback
}

// Listen starts network server
func (s *server) Listen(packge_type string) {
	var listener net.Listener
	var err error
	if s.config == nil {
		listener, err = net.Listen("tcp", s.address)
	} else {
		listener, err = tls.Listen("tcp", s.address, s.config)
	}
	if err != nil {
		log.Fatal("Error starting TCP server.\r\n", err)
	}
	defer listener.Close()

	for {
		conn, _ := listener.Accept()
		client := &Client{
			conn:   conn,
			Server: s,
		}
		go client.listen(packge_type)
	}
}

// Creates new tcp server instance
func New(address string) *server {
	log.Println("Creating server with address", address)
	server := &server{
		address: address,
	}

	server.OnNewClient(func(c *Client) {})
	server.OnNewMessage(func(c *Client, message []byte) {})
	server.OnClientConnectionClosed(func(c *Client, err error) {})

	return server
}

func NewWithTLS(address, certFile, keyFile string) *server {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal("Error loading certificate files. Unable to create TCP server with TLS functionality.\r\n", err)
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	server := New(address)
	server.config = config
	return server
}
