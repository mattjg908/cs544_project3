package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"drexel.edu/net-quic/pkg/pdu"
	"drexel.edu/net-quic/pkg/util"
	"github.com/quic-go/quic-go"
)

type ClientConfig struct {
	ServerAddr string
	PortNumber int
	CertFile   string
}

type Client struct {
	cfg      ClientConfig
	tls      *tls.Config
	conn     quic.Connection
	ctx      context.Context
	nickname string
	away     bool
	authed   bool
}

func NewClient(cfg ClientConfig) *Client {
	cli := &Client{
		cfg: cfg,
	}

	if cfg.CertFile != "" {
		log.Printf("[cli] using cert file: %s", cfg.CertFile)
		t, err := util.BuildTLSClientConfigWithCert(cfg.CertFile)
		if err != nil {
			log.Fatal("[cli] error building TLS client config:", err)
			return nil
		}
		cli.tls = t
	} else {
		cli.tls = util.BuildTLSClientConfig()
	}

	cli.ctx = context.TODO()
	return cli
}

func (c *Client) Run(mtype uint8, s string) error {
	serverAddr := fmt.Sprintf("%s:%d", c.cfg.ServerAddr, c.cfg.PortNumber)
	conn, err := quic.DialAddr(c.ctx, serverAddr, c.tls, nil)
	if err != nil {
		log.Printf("[cli] error dialing server %s", err)
		return err
	}
	c.conn = conn
	return c.protocolHandler(mtype, s)
}

func (c *Client) protocolHandler(mtype uint8, s string) error {
	// The first bit of code is only related to connecting to a server. Once
	// that connection is made, we can listen for messages and take user IO

	// Start connecting to server
	stream, err := c.conn.OpenStreamSync(c.ctx)
	if err != nil {
		log.Printf("[cli] error opening stream %s", err)
		return err
	}

	req := pdu.NewPDU(mtype, []byte(s))
	pduBytes, err := pdu.PduToBytes(req)
	if err != nil {
		log.Printf("[cli] error making pdu byte array %s", err)
		return err
	}

	n, err := stream.Write(pduBytes)
	if err != nil {
		log.Printf("[cli] error writing to stream %s", err)
		return err
	}

	buffer := pdu.MakePduBuffer()
	n, err = stream.Read(buffer)
	if err != nil {
		log.Printf("[cli] error reading from stream %s", err)
		return err
	}
	rsp, err := pdu.PduFromBytes(buffer[:n])
	if err != nil {
		log.Printf("[cli] error converting pdu from bytes %s", err)
		return err
	}
	// connecting to server complete

	// naive auth approach, naturally this would be different in a "real" app. If
	// we got here, the password was correct when we attempted server connection
	// Being auth'd is part of the DFA
	c.authed = true

	// we get the nickname back from server once we connect, we track this so we
	// can include our nickname when DMing another client, that way the recipient
	// knows who sent it the message
	c.nickname = string(rsp.Data)

	// listen in the background for direct messages
	go c.ListenForDirectMessages(stream, buffer)

	// ping server in the background to keep connection alive
	go pingServer(stream)

	// start of user chat input
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Enter messages to send to the server. Type 'exit' to quit.")
	for scanner.Scan() {
		msg := scanner.Text()
		switch msg {
		case "exit":
			return stream.Close()
		case "list":
			c.writeIoToStream(stream, pdu.TYPE_LIST, "")
		case "away":
			c.away = !c.away
			// direct messages
		default:
			c.writeIoToStream(stream, pdu.TYPE_DM, msg+"|"+c.nickname)
		}

	}
	// end of user input

	return stream.Close()
}

func (c *Client) writeIoToStream(stream quic.Stream, pduType uint8, msg string) {
	c.checkIsAuthed(stream)

	req := pdu.NewPDU(pduType, []byte(msg))
	pduBytes, err := pdu.PduToBytes(req)

	if err != nil {
		log.Printf("[cli] error making pdu byte array %s", err)
	}

	stream.Write(pduBytes)
}

func (c *Client) checkIsAuthed(stream quic.Stream) {
	if !c.authed {
		log.Printf("Not authorized")
		stream.Close()
	}
}

func (c *Client) ListenForDirectMessages(stream quic.Stream, buffer []byte) {
	for {
		n, err := stream.Read(buffer)
		if err != nil {
			log.Printf("error reading from stream: %s", err)
			break
		}
		rsp, err := pdu.PduFromBytes(buffer[:n])
		if err != nil {
			log.Printf("[cli] error converting pdu from bytes %s", err)
			continue
		}
		rspDataString := string(rsp.Data)
		log.Printf(rspDataString)

		if c.away {
			// Parse out sender's name, so we can reply "I am away" to them
			params := strings.Split(string(rspDataString), ":")

			c.writeIoToStream(stream, pdu.TYPE_DM, params[0]+"|"+"I am away"+"|"+c.nickname)
		}
	}
}

// keep connection to server alive
func pingServer(stream quic.Stream) {
	for {
		req := pdu.NewPDU(pdu.TYPE_PING, []byte(""))
		pduBytes, _ := pdu.PduToBytes(req)
		stream.Write(pduBytes)
		time.Sleep(20 * time.Second)
	}
}
