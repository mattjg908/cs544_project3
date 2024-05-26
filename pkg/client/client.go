package client

import (
  "bufio"
	"context"
	"crypto/tls"
	"fmt"
	"log"
  "os"

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
	cfg  ClientConfig
	tls  *tls.Config
	conn quic.Connection
	ctx  context.Context
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
	log.Printf("[cli] wrote %d bytes to stream", n)

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
	rspDataString := string(rsp.Data)
	log.Printf("[cli] got response: %s", rsp.ToJsonString())
	log.Printf("[cli] decoded string: %s", rspDataString)

  // start of user input
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Enter messages to send to the server. Type 'exit' to quit.")
	for scanner.Scan() {
		msg := scanner.Text()
	  switch msg {
	  case "exit":
	    return stream.Close()
	  case "list":
	    req := pdu.NewPDU(pdu.TYPE_LIST, []byte(""))
	    pduBytes, err := pdu.PduToBytes(req)
	    if err != nil {
	    	log.Printf("[cli] error making pdu byte array %s", err)
	    	return err
	    }
	    stream.Write(pduBytes)
	  default:
	  	// continue listening to input
	  }

  }
  // end of user input


  //return nil
	return stream.Close()
}
