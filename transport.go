package blynk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	slog "github.com/OloloevReal/go-simple-log"
)

func (g *Blynk) sendMessage(msg BlynkMessage) (uint16, error) {
	if err := g.sendBytes(msg.GetBytes()); err != nil {
		return 0, err
	}
	return msg.Head.MessageId, nil
}

func (g *Blynk) sendCommand(cmd BlynkCommand) (uint16, error) {
	msg := BlynkMessage{}
	msg.Head.Command = cmd
	msg.Head.MessageId = g.getMessageID()
	msg.Head.Length = 0
	return g.sendMessage(msg)
}

func (g *Blynk) sendString(cmd BlynkCommand, data string) (uint16, error) {
	if g.conn == nil {
		return 0, fmt.Errorf("send: conn *net.TCPConn is nil")
	}

	msg := BlynkMessage{}
	msg.Head.Command = cmd
	msg.Head.MessageId = g.getMessageID()
	msg.Body.AddString(data)
	msg.Head.Length = msg.Body.Len()

	err := g.sendBytes(msg.GetBytes())
	if err != nil {
		return msg.Head.MessageId, err
	}

	return msg.Head.MessageId, nil
}

func (g *Blynk) sendBytes(buf []byte) error {
	_, err := g.conn.Write(buf)
	return err
}

func (g *Blynk) receiveMessage(timeout time.Duration) (*BlynkHead, error) {
	buf, err := g.receive(timeout)
	if err != nil {
		return nil, err
	}
	resp := new(BlynkHead)

	bufReader := bytes.NewBuffer(buf)

	err = binary.Read(bufReader, binary.BigEndian, resp)
	if err != nil {
		slog.Printf("[DEBUG] receiveMessage: binary read error, ", err.Error())
		return nil, err
	}

	return resp, nil

}

func (g *Blynk) receive(timeout time.Duration) ([]byte, error) {
	if g == nil || g.conn == nil {
		return nil, fmt.Errorf("receive: *Blynk or *net.Conn is nil")
	}

	g.conn.SetDeadline(time.Now().Add(timeout))
	defer g.conn.SetDeadline(time.Time{})

	buf := make([]byte, 1024)
	cnt, err := g.conn.Read(buf)
	if err == io.EOF {
		slog.Printf("[DEBUG] receive: EOF")
		return nil, err
	}

	if err2, ok := err.(net.Error); ok && err2.Timeout() {
		slog.Printf("[DEBUG] is timeout: %v %d\n", err2.Timeout(), cnt)
		return nil, err2
	}

	if err != nil {
		slog.Printf("[DEBUG] receive: error, ", err.Error())
		return nil, err
	}

	return buf, nil
}

func (g *Blynk) receiver() error {
	slog.Printf("[INFO] Receiver: started")
	defer slog.Printf("[INFO] Receiver: finished")
	if g == nil || g.conn == nil {
		return fmt.Errorf("receiver: *Blynk or *net.TCPConn is nil")
	}
	g.conn.SetReadDeadline(time.Time{})
	buf := make([]byte, 1024)
	for {
		select {
		case <-g.cancel:
			slog.Printf("[DEBUG] receiver: cancel received")
			return nil
		default:
			{
				cntBytes, err := g.conn.Read(buf)
				if err == io.EOF {
					slog.Printf("[DEBUG] receiver: EOF")
					break
				}
				if err2, ok := err.(net.Error); ok && err2.Timeout() {
					slog.Printf("[DEBUG] receiver: is timeout: %v\n", err2.Timeout())
					break
				}
				if err != nil {
					slog.Printf("[ERROR] receiver: error, %s", err.Error())
					return err
				}
				//slog.Printf("[DEBUG] receiver send: % x", buf[:cntBytes])
				bufToSend := make([]byte, cntBytes)
				copy(bufToSend, buf[:cntBytes])
				g.recvMsg <- bufToSend
			}
		}
	}

	return nil
}

func (g *Blynk) processor() {
	slog.Printf("Processor: started")
	defer slog.Printf("Processor: finished")
	for {
		select {
		case <-g.cancel:
			slog.Printf("[DEBUG] Processor: Stop received")
			return
		case buf := <-g.recvMsg:
			{
				//slog.Printf("[DEBUG] processor received msg: % x", buf)
				br, err := g.parseResponce(buf)
				if err != nil {
					slog.Printf("[ERROR] processor: error parsing, %s", err.Error())
				}

				for _, resp := range br {
					switch resp.Command {
					case BLYNK_CMD_HARDWARE:
						if g.OnReadFunc != nil {
							g.OnReadFunc(resp)
						}

						switch resp.Values[0] {
						case "vr":
							pin, _ := strconv.Atoi(resp.Values[1])
							if reader, ok := g.readers[uint(pin)]; !ok {
								slog.Printf("[DEBUG] failed to find reader, Pin: %d", pin)
							} else {
								var buf bytes.Buffer
								reader(uint(pin), &buf)
								slog.Printf("[DEBUG] reader result: %s", buf.String())
								g.VirtualWrite(pin, buf.String())
							}
						case "vw":
							pin, _ := strconv.Atoi(resp.Values[1])
							if writer, ok := g.writers[uint(pin)]; !ok {
								slog.Printf("[DEBUG] failed to find reader, Pin: %d", pin)
							} else {
								var buf bytes.Buffer
								buf.WriteString(resp.Values[2])
								slog.Printf("[DEBUG] value: %s", resp.Values[2])
								writer(uint(pin), &buf)
							}
						}

					case BLYNK_CMD_RESPONSE:

					case BLYNK_CMD_PING:
						g.sendPingResponse(resp.MessageId)
					default:
						slog.Printf("[ERROR] Processor received unhandled msg: %v", resp)
					}
				}

			}
		}
	}

}

func (g *Blynk) parseResponce(buf []byte) ([]*BlynkRespose, error) {
	var resps []*BlynkRespose
	var resp *BlynkRespose
	flagStart := 0
	if len(buf) >= 5 {
		for len(buf) >= flagStart+5 {
			resp = new(BlynkRespose)
			resp.parseHead(buf[flagStart : flagStart+5])
			lenBody := int(resp.Status)

			if resp.Command == BLYNK_CMD_HARDWARE && resp.Status > 0 && resp.Status < 1024 {
				if len(buf) >= flagStart+5+lenBody {
					resp.parseBody(buf[flagStart+5 : flagStart+5+lenBody])
				} else {
					slog.Printf("[ERROR] parseResponce: failed parse body, length of buf less than flag")
				}

			}

			resps = append(resps, resp)
			flagStart += 5 + lenBody
		}
	}

	return resps, nil
}

func (g *Blynk) sendPing() error {
	if _, err := g.sendCommand(BLYNK_CMD_PING); err != nil {
		return err
	}
	return nil
}

func (g *Blynk) sendPingResponse(id uint16) error {
	msg := BlynkMessage{}
	msg.Head.Command = BLYNK_CMD_RESPONSE
	msg.Head.MessageId = id
	msg.Head.Length = BLYNK_SUCCESS

	if _, err := g.sendMessage(msg); err != nil {
		return err
	}

	return nil
}
