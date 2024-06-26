package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	"drexel.edu/net-quic/pkg/pdu"
	"drexel.edu/net-quic/pkg/util"
	"github.com/quic-go/quic-go"
)

const (
	// Temp. hardcoded values
	PASSWORD = "password123"
)

type contextKey string

const nicknamesKey contextKey = "nicknames"

type ServerConfig struct {
	GenTLS   bool
	CertFile string
	KeyFile  string
	Address  string
	Port     int
	PeerAddr string
	PeerPort int
}

type Server struct {
	cfg        ServerConfig
	tls        *tls.Config
	ctx        context.Context
	clients    map[string]quic.Stream
	peerConn   quic.Connection
	peerStream quic.Stream
}

func NewServer(cfg ServerConfig) *Server {
	server := &Server{
		cfg:     cfg,
		clients: make(map[string]quic.Stream),
	}
	server.tls = server.getTLS()
	// so we can administrate the connected clients
	server.ctx = context.WithValue(context.Background(), nicknamesKey, []string{})
	return server
}

func (s *Server) getTLS() *tls.Config {
	if s.cfg.GenTLS {
		tlsConfig, err := util.GenerateTLSConfig()
		if err != nil {
			log.Fatal(err)
		}
		return tlsConfig
	} else {
		tlsConfig, err := util.BuildTLSConfig(s.cfg.CertFile, s.cfg.KeyFile)
		if err != nil {
			log.Fatal(err)
		}
		return tlsConfig
	}
}

func (s *Server) Run() error {
	address := fmt.Sprintf("%s:%d", s.cfg.Address, s.cfg.Port)
	listener, err := quic.ListenAddr(address, s.tls, nil)
	if err != nil {
		log.Printf("error listening: %s", err)
		return err
	}

	go s.ConnectToPeerServer()

	//SERVER LOOP
	for {
		log.Println("Accepting new session")
		sess, err := listener.Accept(s.ctx)
		if err != nil {
			log.Printf("error accepting: %s", err)
			return err
		}

		go s.streamHandler(sess)
	}
}

func (s *Server) streamHandler(sess quic.Connection) {
	for {
		log.Print("[server] waiting for client to open stream")
		stream, err := sess.AcceptStream(s.ctx)
		if err != nil {
			log.Printf("[server] stream closed: %s", err)
			break
		}

		//Handle protocol activity on stream
		s.protocolHandler(stream)
	}
}

func (s *Server) protocolHandler(stream quic.Stream) error {
	for {
		//THIS IS WHERE YOU START HANDLING YOUR APP PROTOCOL
		buff := pdu.MakePduBuffer()

		n, err := stream.Read(buff)
		if err != nil {
			log.Printf("[server] Error Reading Raw Data: %s", err)
			return err
		}

		data, err := pdu.PduFromBytes(buff[:n])
		if err != nil {
			log.Printf("[server] Error decoding PDU: %s", err)
			return err
		}

		// Split the data out so we can parse it
		params := strings.Split(string(data.Data), "|")

		switch data.Mtype {
		case pdu.TYPE_CLIENT_CONNECT:
			if params[1] == "password123" {
				fmt.Println("Password is correct")
				s.addClient(params[0], stream)

				// return the nickname so the client can store it, this is used for
				// so the client can send a message to another client and the sender
				// can be tracked
				rspPdu := pdu.PDU{
					Mtype: pdu.TYPE_DATA | pdu.TYPE_ACK,
					Len:   uint32(len(params[0])),
					Data:  []byte(params[0]),
				}

				fmt.Printf("Server-> %v", rspPdu)

				rspBytes, err := pdu.PduToBytes(&rspPdu)
				if err != nil {
					log.Printf("[server] Error encoding PDU: %s", err)
					return err
				}

				_, err = stream.Write(rspBytes)
				if err != nil {
					log.Printf("[server] Error sending response: %s", err)
					return err
				}
			} else {
				// Close connection if password is wrong
				fmt.Println("incorrect or unknown credentials")
				return stream.Close()
			}
			// List nicknames on peer servers
		case pdu.TYPE_PEER_LIST:
			nicknames := s.getNicknames()

			nicknamesData := strings.Join(nicknames, ",")

			rspPdu := pdu.PDU{
				Mtype: pdu.TYPE_DATA,
				Len:   uint32(len(nicknamesData)),
				Data:  []byte(nicknamesData),
			}

			rspBytes, err := pdu.PduToBytes(&rspPdu)
			if err != nil {
				log.Printf("[server] Error encoding PDU: %s", err)
				break
			}
			stream.Write(rspBytes)

		case pdu.TYPE_LIST:
			nicknames := s.getNicknames()

			nicknamesData := strings.Join(nicknames, ",")

			rspPdu := pdu.PDU{
				Mtype: pdu.TYPE_PEER_LIST,
				Len:   uint32(len("")),
				Data:  []byte(""),
			}

			rspBytes, err := pdu.PduToBytes(&rspPdu)
			if err != nil {
				log.Printf("[server] Error encoding PDU: %s", err)
				break
			}

			// if connected to a peer server, list connected clients there too
			if s.peerStream != nil {
				s.peerStream.Write(rspBytes)
				n, err := s.peerStream.Read(buff)
				if err != nil {
					log.Printf("[server] Error Reading Raw Data: %s", err)
					return err
				}

				peerData, err := pdu.PduFromBytes(buff[:n])

				nicknamesData = nicknamesData + "," + string(peerData.Data)
			}

			rspPdu = pdu.PDU{
				Mtype: pdu.TYPE_DATA,
				Len:   uint32(len(nicknamesData)),
				Data:  []byte(nicknamesData),
			}

			rspBytes, err = pdu.PduToBytes(&rspPdu)
			if err != nil {
				log.Printf("[server] Error encoding PDU: %s", err)
				break
			}
			stream.Write(rspBytes)

		case pdu.TYPE_DM:
			s.sendPrivateMessage(params[0], params[1], params[2])

		default:
			continue
		}

		log.Printf("[server] Data In: [%s]", data.GetTypeAsString())
	}
}

/*
  SOME functions below are modified from ChatGPT and/or examples found online
*/

func (s *Server) addClient(nickname string, stream quic.Stream) {
	s.clients[nickname] = stream
	s.addNickname(nickname)
}

func (s *Server) sendPrivateMessage(recipient, message string, sender string) {

	stream, exists := s.clients[recipient]
	if !exists {
		log.Printf("[server] Recipient %s not found", recipient)

		// Reconstruct message, send it to other server(s)
		rspPdu := pdu.PDU{
			Mtype: pdu.TYPE_DM,
			Len:   uint32(len(recipient + "|" + message + "|" + sender)),
			Data:  []byte(recipient + "|" + message + "|" + sender),
		}

		rspBytes, _ := pdu.PduToBytes(&rspPdu)
		s.peerStream.Write(rspBytes)
		return
	}

	rspPdu := pdu.PDU{
		Mtype: pdu.TYPE_DATA,
		Len:   uint32(len(sender + ": " + message)),
		Data:  []byte(sender + ": " + message),
	}

	rspBytes, err := pdu.PduToBytes(&rspPdu)
	if err != nil {
		log.Printf("[server] Error encoding PDU: %s", err)
		return
	}

	_, err = stream.Write(rspBytes)
	if err != nil {
		log.Printf("[server] Error sending private message to %s: %s", recipient, err)
		return
	}

	log.Printf("[server] Private message sent to %s: %s", recipient, message)
}

// Just some helper functions to track connected clients
func (s *Server) addNickname(nickname string) {
	s.ctx = s.updateNicknamesContext(nickname, true)
}

func (s *Server) removeNickname(nickname string) {
	s.ctx = s.updateNicknamesContext(nickname, false)
}

func (s *Server) updateNicknamesContext(nickname string, add bool) context.Context {
	nicknames := s.getNicknames()
	if add {
		nicknames = append(nicknames, nickname)
	} else {
		for i, n := range nicknames {
			if n == nickname {
				nicknames = append(nicknames[:i], nicknames[i+1:]...)
				break
			}
		}
	}
	return context.WithValue(s.ctx, nicknamesKey, nicknames)
}

func (s *Server) getNicknames() []string {
	return s.ctx.Value(nicknamesKey).([]string)
}

func (s *Server) ConnectToPeerServer() error {
	// hardcoded peer ports
	var port int
	if s.cfg.Port == 4242 {
		port = 4243
	} else {
		port = 4242
	}
	peerAddr := fmt.Sprintf("%s:%d", "localhost", port)
	tlsConfig := util.BuildTLSClientConfig()

	conn, err := quic.DialAddr(s.ctx, peerAddr, tlsConfig, nil)
	if err != nil {
		log.Printf("[server] error dialing peer server: %s, have you started peer on port %d?", err, port)
		time.Sleep(2 * time.Second)
		// try again
		s.ConnectToPeerServer()
		return err
	}

	stream, err := conn.OpenStreamSync(s.ctx)
	if err != nil {
		log.Printf("[server] error opening stream to peer server: %s", err)
		return err
	}

	s.peerConn = conn
	s.peerStream = stream
	log.Printf("[server] connected to peer server at %s", peerAddr)
	// ping server to keep connection alive
	go pingServer(stream)
	return nil
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
