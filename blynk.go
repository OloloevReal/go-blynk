package blynk

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	certs "github.com/OloloevReal/go-blynk/certs"
)

const Version = "0.0.3"

type Blynk struct {
	APIkey     string
	server     string
	port       int
	OnReadFunc func(*BlynkRespose)
	conn       net.Conn
	msgID      uint16
	heartbeat  time.Duration
	timeout    time.Duration
	timeoutMAX time.Duration
	lock       sync.Mutex
	ssl        bool
	cancel     chan bool
}

func NewBlynk(APIkey string, Server string, Port int, SSL bool) *Blynk {
	return &Blynk{APIkey: APIkey,
		server:     Server,
		port:       Port,
		conn:       nil,
		msgID:      0,
		heartbeat:  time.Second * 10,
		timeout:    time.Millisecond * 50,
		timeoutMAX: time.Second * 5,
		lock:       sync.Mutex{},
		ssl:        SSL,
		cancel:     make(chan bool, 1),
	}
}

func (g *Blynk) printLogo() {
	logo := `
     ___  __          __
    / _ )/ /_ _____  / /__
   / _  / / // / _ \/  '_/
  /____/_/\_, /_//_/_/\_\
         /___/ for Go v%s (%s)


`
	fmt.Printf(logo, Version, runtime.GOOS)
}

func (g *Blynk) Connect() error {

	g.printLogo()

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", g.server, g.port))
	if err != nil {
		return err
	}

	if g.ssl {
		g.conn, err = g.dialTLS(addr)
	} else {
		g.conn, err = net.DialTCP("tcp", nil, addr)
	}

	if err != nil {
		return err
	}
	//defer conn.Close()

	if err = g.auth(); err != nil {
		return err
	}
	log.Printf("Connect: Auth success (SSL: %v)", g.ssl)

	g.sendInternal()
	return nil
}

func (g *Blynk) dialTLS(addr *net.TCPAddr) (*tls.Conn, error) {
	roots := x509.NewCertPool()
	rootPEM, err := g.loadCA()
	if err != nil {
		return nil, err
	}
	ok := roots.AppendCertsFromPEM(rootPEM)
	if !ok {
		return nil, fmt.Errorf("failed to parse root certificate")
	}

	//w := os.Stdout
	conf := tls.Config{
		InsecureSkipVerify:     false,
		MinVersion:             tls.VersionTLS12,
		RootCAs:                roots,
		ServerName:             g.server,
		SessionTicketsDisabled: true,
		//KeyLogWriter:           w,
	}
	conn, err := tls.Dial("tcp", addr.String(), &conf)
	return conn, err
}

func (g *Blynk) loadCA() ([]byte, error) {
	return []byte(certs.CertServer), nil
}

func (g *Blynk) Processing() {
	go g.keepAlive()
	g.receiver()
}

func (g *Blynk) getMessageID() uint16 {
	g.lock.Lock()
	defer g.lock.Unlock()
	g.msgID++
	if g.msgID > 0xFFFF {
		g.msgID = 1
	}
	return g.msgID
}

func (g *Blynk) auth() error {
	err := g.send(BLYNK_CMD_HW_LOGIN, g.APIkey)
	if err != nil {
		return err
	}

	response, err := g.receive(g.timeoutMAX)
	if err != nil {
		return err
	}
	//log.Printf("Auth: response: %v", response)
	if response != nil && (response.MessageId != g.msgID || response.Command != BLYNK_CMD_RESPONSE || response.Length != BLYNK_SUCCESS) {
		return fmt.Errorf("auth: failed, message id-%d, code-%d", response.MessageId, response.Length)
	}
	return nil
}

func (g *Blynk) VirtualWrite(pin int, value string) error {

	b := strings.Builder{}
	b.Write([]byte("vw"))
	b.WriteByte(0x00)

	b.Write([]byte(strconv.Itoa(pin)))
	b.WriteByte(0x00)
	b.Write([]byte(value))
	if err := g.send(BLYNK_CMD_HARDWARE, b.String()); err != nil {
		return err
	}
	return nil
}

func (g *Blynk) VirtualRead(pins ...int) error {

	b := strings.Builder{}
	b.Write([]byte("vr"))
	b.WriteByte(0x00)
	for _, v := range pins {
		b.Write([]byte(strconv.Itoa(v)))
		b.WriteByte(0x00)
	}

	if err := g.send(BLYNK_CMD_HARDWARE_SYNC, b.String()); err != nil {
		return err
	}

	return nil
}

func (g *Blynk) DigitalWrite(pin int, value bool) error {
	b := strings.Builder{}
	b.Write([]byte("dw"))
	b.WriteByte(0x00)

	b.Write([]byte(strconv.Itoa(pin)))
	b.WriteByte(0x00)
	if value {
		b.WriteByte(0x01)
	} else {
		b.WriteByte(0x00)
	}

	if err := g.send(BLYNK_CMD_HARDWARE, b.String()); err != nil {
		return err
	}
	return nil
}

func (g *Blynk) DigitalRead(pin int) error {
	b := strings.Builder{}
	b.Write([]byte("dr"))
	b.WriteByte(0x00)
	b.Write([]byte(strconv.Itoa(pin)))
	b.WriteByte(0x00)

	if err := g.send(BLYNK_CMD_HARDWARE_SYNC, b.String()); err != nil {
		return err
	}

	return nil
}

func (g *Blynk) send(cmd BlynkCommand, data string) error {
	if g.conn == nil {
		return fmt.Errorf("send: conn *net.TCPConn is nil")
	}

	var writer bytes.Buffer
	msg := BlynkHead{}
	msg.Command = cmd
	msg.MessageId = g.getMessageID()

	msg.Length = uint16(len(data))

	err := binary.Write(&writer, binary.BigEndian, msg)
	if err != nil {
		return err
	}

	if len(data) > 0 {
		err = binary.Write(&writer, binary.BigEndian, []byte(data))
		if err != nil {
			return err
		}
	}

	err = g.sendBytes(writer.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (g *Blynk) sendCMD(cmd *BlynkHead) error {
	var writer bytes.Buffer
	err := binary.Write(&writer, binary.BigEndian, cmd)
	if err != nil {
		return err
	}

	err = g.sendBytes(writer.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (g *Blynk) sendBytes(buf []byte) error {
	_, err := g.conn.Write(buf)
	return err
}

func (g *Blynk) receive(timeout time.Duration) (*BlynkHead, error) {
	if g == nil || g.conn == nil {
		return nil, fmt.Errorf("reveive: *Glynk or *net.Conn is nil")
	}

	resp := new(BlynkHead)
	g.conn.SetDeadline(time.Now().Add(timeout))
	defer g.conn.SetDeadline(time.Time{})

	buf := make([]byte, 1024)
	cnt, err := g.conn.Read(buf)
	if err == io.EOF {
		log.Println("receive: EOF")
		return nil, err
	}

	if err2, ok := err.(net.Error); ok && err2.Timeout() {
		log.Printf("is timeout: %v %d\n", err2.Timeout(), cnt)
		return nil, nil
	}

	if err != nil {
		log.Println("receive: error, ", err.Error())
		return nil, err
	}

	bufReader := bytes.NewBuffer(buf)

	err = binary.Read(bufReader, binary.BigEndian, resp)
	if err != nil {
		log.Println("receive: binary read error, ", err.Error())
		return nil, err
	}

	return resp, nil
}

func (g *Blynk) receiver() error {
	log.Println("Receiver: started")
	defer log.Println("Receiver: finished")
	if g == nil || g.conn == nil {
		return fmt.Errorf("receiver: *Glynk or *net.TCPConn is nil")
	}
	g.conn.SetReadDeadline(time.Time{})
	buf := make([]byte, 1024)
	for {
		select {
		case <-g.cancel:
			log.Println("receiver: cancel received")
			return nil
		default:
			{
				cntBytes, err := g.conn.Read(buf)
				if err == io.EOF {
					log.Println("receiver: EOF")
					break
				}
				if err2, ok := err.(net.Error); ok && err2.Timeout() {
					log.Printf("receiver: is timeout: %v\n", err2.Timeout())
					break
				}
				if err != nil {
					log.Printf("receiver: error, %s", err.Error())
					return err
				}
				//log.Printf("Receiver: % x - %s", buf[:cntBytes], string(buf[:cntBytes]))
				br, err := g.parseResponce(buf[:cntBytes])
				if err != nil {
					log.Printf("receiver: error parsing, %s", err.Error())
				}
				//log.Printf("%#v", br)

				for _, resp := range br {

					switch resp.Command {
					case BLYNK_CMD_HARDWARE:
						if resp.Command == BLYNK_CMD_HARDWARE {
							if g.OnReadFunc != nil {
								g.OnReadFunc(resp)
							}
						} else {
							log.Printf("Receiver: %v", resp)
						}
					case BLYNK_CMD_PING:
						g.sendPingResponse(resp.MessageId)
					}

				}
			}
		}
	}

	return nil
}

func (g *Blynk) parseResponce(buf []byte) ([]*BlynkRespose, error) {
	var resps []*BlynkRespose
	var resp *BlynkRespose
	flag_start := 0
	if len(buf) >= 5 {
		for len(buf) >= flag_start+5 {
			resp = new(BlynkRespose)
			resp.parseHead(buf[flag_start : flag_start+5])
			lenBody := int(resp.Status)

			if resp.Command == BLYNK_CMD_HARDWARE && resp.Status > 0 && resp.Status < 1024 {
				if len(buf) >= flag_start+5+lenBody {
					resp.parseBody(buf[flag_start+5 : flag_start+5+lenBody])
				} else {
					log.Println("parseResponce: failed parse body, length of buf less than flag")
				}

			}

			resps = append(resps, resp)
			flag_start += 5 + lenBody
		}
	}

	return resps, nil
}

func (g *Blynk) sendPing() error {

	if err := g.send(BLYNK_CMD_PING, ""); err != nil {
		return err
	}

	return nil
}

func (g *Blynk) sendPingResponse(id uint16) error {

	cmd := &BlynkHead{
		Command:   BLYNK_CMD_RESPONSE,
		MessageId: id,
		Length:    BLYNK_SUCCESS,
	}
	if err := g.sendCMD(cmd); err != nil {
		return err
	}

	return nil
}

func (g *Blynk) sendInternal() error {
	if err := g.send(BLYNK_CMD_INTERNAL, g.formatInternal()); err != nil {
		return err
	}

	resp, err := g.receive(g.timeoutMAX)
	if err != nil {
		return err
	}

	if resp.Length != BLYNK_SUCCESS {
		return fmt.Errorf("sendInternal: received unsuccessful code %d", resp.Length)
	}

	return nil
}

func (g *Blynk) formatInternal() string {
	rcv_buffer := "1024"
	params := []string{"ver", Version, "buff-in", rcv_buffer, "h-beat", fmt.Sprintf("%.0f", g.heartbeat.Seconds()), "dev", "go"}
	return strings.Join(params, string(0x00))
}

func (g *Blynk) keepAlive() {
	log.Println("Keep-Alive: started")
	defer log.Println("Keep-Alive: finished")
	t := time.NewTicker(g.heartbeat)
	for {
		select {
		case <-t.C:
			//log.Println("Keep-Alive")
			g.send(BLYNK_CMD_PING, "")
		case <-g.cancel:
			t.Stop()
			return
		}
	}
}

func (g *Blynk) Stop() error {
	if g == nil {
		return fmt.Errorf("Glynk: source object glynk is nil")
	}
	log.Println("Sending to cancle channel")
	g.conn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
	close(g.cancel)
	time.Sleep(time.Second * 1)
	return g.Disconnect()
}

func (g *Blynk) Disconnect() error {
	if g == nil || g.conn == nil {
		return fmt.Errorf("disconnect: *Glynk or *net.TCPConn is nil")
	}
	err := g.conn.Close()
	return err
}
