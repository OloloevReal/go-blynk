package blynk

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"strings"
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
type BlynkBody strings.Builder

type BlynkCommand byte

const (
	BLYNK_CMD_RESPONSE      BlynkCommand = 0
	BLYNK_CMD_LOGIN         BlynkCommand = 2
	BLYNK_CMD_PING          BlynkCommand = 6
	BLYNK_CMD_TWEET         BlynkCommand = 12
	BLYNK_CMD_EMAIL         BlynkCommand = 13
	BLYNK_CMD_NOTIFY        BlynkCommand = 14
	BLYNK_CMD_HARDWARE_SYNC BlynkCommand = 16
	BLYNK_CMD_INTERNAL      BlynkCommand = 17
	BLYNK_CMD_HARDWARE      BlynkCommand = 20
	BLYNK_CMD_HW_LOGIN      BlynkCommand = 29
)

const (
	BLYNK_SUCCESS             uint16 = 200
	BLYNK_ILLEGAL_COMMAND     uint16 = 2
	BLYNK_NOT_REGISTERED      uint16 = 3
	BLYNK_NOT_ALLOWED         uint16 = 6
	BLYNK_NO_ACTIVE_DASHBOARD uint16 = 8
	BLYNK_INVALID_TOKEN       uint16 = 9
	BLYNK_NTF_INVALID_BODY    uint16 = 13
	BLYNK_NTF_NOT_AUTHORIZED  uint16 = 14
	BLYNK_NTF_EXCEPTION       uint16 = 15
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

func GetBlynkStatus(status uint16) string {
	switch status {
	case BLYNK_SUCCESS:
		return "SUCCESS"
	case BLYNK_ILLEGAL_COMMAND:
		return "ILLEGAL_COMMAND"
	case BLYNK_NOT_REGISTERED:
		return "NOT_REGISTERED"
	case BLYNK_NOT_ALLOWED:
		return "NOT_ALLOWED"
	case BLYNK_NO_ACTIVE_DASHBOARD:
		return "NO_ACTIVE_DASHBOARD"
	case BLYNK_INVALID_TOKEN:
		return "INVALID_TOKEN"
	case BLYNK_NTF_INVALID_BODY:
		return "NTF_INVALID_BODY"
	case BLYNK_NTF_NOT_AUTHORIZED:
		return "NTF_NOT_AUTHORIZED"
	case BLYNK_NTF_EXCEPTION:
		return "NTF_EXCEPTION"
	default:
		return "UNDEFINED"
	}
}

func (b *BlynkBody) String() string {
	if b == nil {
		return ""
	}

	builder := (*strings.Builder)(b)
	return builder.String()
}

func (b *BlynkBody) Clear() {
	if b == nil {
		return
	}
	builder := (*strings.Builder)(b)
	builder.Reset()
}

func (b *BlynkBody) AddString(s string) {
	if b == nil {
		return
	}
	builder := (*strings.Builder)(b)
	if builder.Len() != 0 {
		builder.WriteByte(0x00)
	}
	builder.Write([]byte(s))

}

func (b *BlynkBody) AddInt(values ...int) {
	if b == nil {
		return
	}
	builder := (*strings.Builder)(b)
	for _, v := range values {
		if builder.Len() != 0 {
			builder.WriteByte(0x00)
		}
		builder.Write([]byte(strconv.Itoa(v)))
	}
}

func (b *BlynkBody) AddBool(v bool) {
	if b == nil {
		return
	}

	builder := (*strings.Builder)(b)
	if builder.Len() != 0 {
		builder.WriteByte(0x00)
	}
	if v {
		builder.WriteByte(0x31)
	} else {
		builder.WriteByte(0x30)
	}
}
