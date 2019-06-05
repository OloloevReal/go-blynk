package blynk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	slog "github.com/OloloevReal/go-simple-log"
)

func (g *Blynk) sendCommand(cmd BlynkCommand) (uint16, error) {
	return g.send(cmd, "")
}

func (g *Blynk) send(cmd BlynkCommand, data string) (uint16, error) {
	if g.conn == nil {
		return 0, fmt.Errorf("send: conn *net.TCPConn is nil")
	}

	var writer bytes.Buffer
	msg := BlynkHead{}
	msg.Command = cmd
	msg.MessageId = g.getMessageID()

	msg.Length = uint16(len(data))

	err := binary.Write(&writer, binary.BigEndian, msg)
	if err != nil {
		return msg.MessageId, err
	}

	if len(data) > 0 {
		err = binary.Write(&writer, binary.BigEndian, []byte(data))
		if err != nil {
			return msg.MessageId, err
		}
	}

	err = g.sendBytes(writer.Bytes())
	if err != nil {
		return msg.MessageId, err
	}

	return msg.MessageId, nil
}

func (g *Blynk) sendPacket(cmd *BlynkHead) error {
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
		return nil, fmt.Errorf("receive: *Blynk or *net.Conn is nil")
	}

	resp := new(BlynkHead)
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

	bufReader := bytes.NewBuffer(buf)

	err = binary.Read(bufReader, binary.BigEndian, resp)
	if err != nil {
		slog.Printf("[DEBUG] receive: binary read error, ", err.Error())
		return nil, err
	}

	return resp, nil
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
					slog.Printf("[DEBUG] receiver: error, %s", err.Error())
					return err
				}
				//log.Printf("Receiver: % x - %s", buf[:cntBytes], string(buf[:cntBytes]))
				br, err := g.parseResponce(buf[:cntBytes])
				if err != nil {
					slog.Printf("[DEBUG] receiver: error parsing, %s", err.Error())
				}

				for _, resp := range br {

					switch resp.Command {
					case BLYNK_CMD_HARDWARE:
						if resp.Command == BLYNK_CMD_HARDWARE {
							if g.OnReadFunc != nil {
								g.OnReadFunc(resp)
							}
						} else {
							slog.Printf("[DEBUG] Receiver: %v", resp)
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
					slog.Printf("[ERROR] parseResponce: failed parse body, length of buf less than flag")
				}

			}

			resps = append(resps, resp)
			flag_start += 5 + lenBody
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

	cmd := &BlynkHead{
		Command:   BLYNK_CMD_RESPONSE,
		MessageId: id,
		Length:    BLYNK_SUCCESS,
	}
	if err := g.sendPacket(cmd); err != nil {
		return err
	}

	return nil
}
