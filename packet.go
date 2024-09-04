package nets

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
)

type Packet struct {
	ID   string `binary:"[24]byte"`
	Type int    `binary:"uint32"`
	Data []byte `binary:"-"`
}

type PacketFuncs interface {
	Encode() ([]byte, error)
	Decode([]byte) error
}

// Ensure Packet implements PacketFuncs interface
var _ PacketFuncs = (*Packet)(nil)

// Encode encodes the Packet into a byte slice using big-endian encoding.
func (p *Packet) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	v := reflect.ValueOf(p).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("binary")

		if tag == "-" {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			size, err := parseSize(tag)
			if err != nil {
				return nil, err
			}
			strBytes := make([]byte, size)
			copy(strBytes, field.String())
			if err := binary.Write(buf, binary.BigEndian, strBytes); err != nil {
				return nil, err
			}
		case reflect.Int:
			if tag == "uint32" {
				if err := binary.Write(buf, binary.BigEndian, uint32(field.Int())); err != nil {
					return nil, err
				}
			}
		case reflect.Slice:
			if fieldType.Type.Elem().Kind() == reflect.Uint8 {
				buf.Write(field.Bytes())
			}
		}
	}

	return buf.Bytes(), nil
}

// Decode decodes a byte slice into the Packet using big-endian encoding.
func (p *Packet) Decode(data []byte) error {
	buf := bytes.NewReader(data)
	v := reflect.ValueOf(p).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("binary")

		if tag == "-" {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			size, err := parseSize(tag)
			if err != nil {
				return err
			}
			strBytes := make([]byte, size)
			if err := binary.Read(buf, binary.BigEndian, &strBytes); err != nil {
				return err
			}
			field.SetString(string(strBytes))
		case reflect.Int:
			if tag == "uint32" {
				var value uint32
				if err := binary.Read(buf, binary.BigEndian, &value); err != nil {
					return err
				}
				field.SetInt(int64(value))
			}
		case reflect.Slice:
			if fieldType.Type.Elem().Kind() == reflect.Uint8 {
				remainingBytes := make([]byte, buf.Len())
				if _, err := buf.Read(remainingBytes); err != nil {
					return err
				}
				field.SetBytes(remainingBytes)
			}
		}
	}

	return nil
}

// parseSize parses the size from the struct tag.
func parseSize(tag string) (int, error) {
	var size int
	_, err := fmt.Sscanf(tag, "[%d]byte", &size)
	if err != nil {
		return 0, errors.New("invalid size tag")
	}
	return size, nil
}
