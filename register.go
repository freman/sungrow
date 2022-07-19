package sungrow

import (
	"bytes"
	"encoding/binary"
	"io"

	"gopkg.in/yaml.v3"
)

type Register struct {
	Address      int                 `yaml:"address"`
	Name         string              `yaml:"name,omitempty"`
	Unit         *string             `yaml:"unit,omitempty"`
	Scale        float64             `yaml:"scale,omitempty"`
	Values       map[int]interface{} `yaml:"values,omitempty"`
	Type         string              `yaml:"type,omitempty"`
	Min          *float64            `yaml:"min,omitempty"`
	Max          *float64            `yaml:"max,omitempty"`
	Count        uint                `yaml:"count,omitempty"`
	Models       Models              `yaml:"models,omitempty"`
	Validity     *string             `yaml:"validity,omitempty"`
	Availibility *string             `yaml:"availibility,omitempty"`

	Value     interface{}
	RAW       []byte
	Err       error
	Supported bool
}

func (r *Register) UnmarshalYAML(value *yaml.Node) error {
	type ttmp Register
	defaults := ttmp{
		Scale: 1.0,
		Type:  "uint16",
		Count: 1,
	}

	if err := value.Decode(&defaults); err != nil {
		return err
	}

	*r = Register(defaults)
	return nil
}

func (r *Register) sizeAs16Bit() int {
	sz := 1

	switch r.Type {
	case "int16", "uint16":
		sz = 1
	case "int32", "uint32":
		sz = 2
	}

	return sz * int(r.Count)
}

func (r *Register) read(rdr io.Reader) (err error) {
	if r.Type == "" {
		r.Type = "uint16"
	}

	var buf bytes.Buffer
	reader := io.TeeReader(rdr, &buf)

	switch r.Type {
	case "int16":
		var read int16
		err = binary.Read(reader, binary.BigEndian, &read)
		r.Value = float64(read) * r.Scale
	case "uint16":
		var read uint16
		err = binary.Read(reader, binary.BigEndian, &read)
		r.Value = float64(read) * r.Scale
	case "int32":
		var read int32
		err = binary.Read(reader, binary.LittleEndian, &read)
		r.Value = float64(read) * r.Scale
	case "uint32":
		var read uint32
		err = binary.Read(reader, binary.LittleEndian, &read)
		r.Value = float64(read) * r.Scale
	case "string":
		b := make([]byte, r.sizeAs16Bit()*2)
		_, err = reader.Read(b)

		for i, v := range b {
			if v == 00 {
				r.Value = string(b[0:i])
				break
			}
		}
	}

	if r.Values != nil {
		switch z := r.Value.(type) {
		case float64:
			r.Value = r.Values[int(z)]
		}
	}
	r.RAW = buf.Bytes()

	if err != nil {
		return err
	}

	return nil
}

func (r *Register) GetUnit() string {
	if r.Unit == nil {
		return ""
	}

	return *r.Unit
}
