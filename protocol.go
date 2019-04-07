package blynk

import (
	"bytes"
	"encoding/binary"
)

type BlynkHead struct {
	Command   BlynkCommand
	MessageId uint16
	Length    uint16
}

type BlynkRespose struct {
	Command   BlynkCommand
	MessageId uint16
	Status    uint16
	Values    []string
}

type BlynkCommand byte

const (
	BLYNK_CMD_RESPONSE      BlynkCommand = 0
	BLYNK_CMD_LOGIN         BlynkCommand = 2
	BLYNK_CMD_PING          BlynkCommand = 6
	BLYNK_CMD_HARDWARE_SYNC BlynkCommand = 16
	BLYNK_CMD_INTERNAL      BlynkCommand = 17
	BLYNK_CMD_HARDWARE      BlynkCommand = 20
	BLYNK_CMD_HW_LOGIN      BlynkCommand = 29
)

const (
	BLYNK_SUCCESS         uint16 = 200
	BLYNK_ILLEGAL_COMMAND uint16 = 2
	BLYNK_NOT_REGISTERED  uint16 = 3
	BLYNK_INVALID_TOKEN   uint16 = 9
)

func (r *BlynkRespose) parseHead(buf []byte) {
	r.Command = BlynkCommand(buf[0])
	r.MessageId = binary.BigEndian.Uint16(buf[1:3])
	r.Status = binary.BigEndian.Uint16(buf[3:5])
	r.Values = nil
}

func (r *BlynkRespose) parseBody(buf []byte) {
	bs := bytes.Split(buf, []byte{0x00})
	for _, s := range bs {
		r.Values = append(r.Values, string(s))
	}
}
