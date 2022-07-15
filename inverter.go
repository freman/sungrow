package sungrow

import (
	"bytes"
	"io"
	"os"

	"github.com/goburrow/modbus"
	"gopkg.in/yaml.v3"
)

type Inverter struct {
	Registers Registers `yaml:"registers,omitempty"`
}

type Modbus interface {
	ReadInputRegisters(address, quantity uint16) (results []byte, err error)
	ReadHoldingRegisters(address, quantity uint16) (results []byte, err error)
}

func (i *Inverter) Define(r io.Reader) error {
	i.Clear()

	return yaml.NewDecoder(r).Decode(i)
}

func (i *Inverter) DefineFromYaml(yamlFile string) error {
	f, err := os.Open(yamlFile)
	if err != nil {
		return err
	}

	defer f.Close()

	return i.Define(f)
}

func (i *Inverter) Clear() {
	i.Registers.Clear()
}

func (i *Inverter) Read(client Modbus) error {
	var model string

	query := []struct {
		code int
		fn   func(address, quantity uint16) (results []byte, err error)
		regs []Register
	}{
		{
			modbus.FuncCodeReadInputRegisters,
			client.ReadInputRegisters,
			i.Registers.Input,
		},
		{
			modbus.FuncCodeReadHoldingRegisters,
			client.ReadHoldingRegisters,
			i.Registers.Holding,
		},
	}

	for _, q := range query {
		for i, v := range q.regs {
			// Model check, don't bother reading registers for models that don't support it
			if model != "" && len(v.Models) > 0 && !v.Models.Contains(model) {
				continue
			}
			v.Supported = true

			results, err := q.fn(uint16(v.Address-1), uint16(v.sizeAs16Bit()))
			if err != nil {
				// Modbus/Register error...
				if e, isa := err.(*modbus.ModbusError); isa {
					v.Err = e
					q.regs[i] = v
					continue
				}

				// Every other error
				return err
			}

			v.read(bytes.NewReader(results))

			if model == "" && v.Address == 5000 {
				switch x := v.Value.(type) {
				case map[string]interface{}:
					if s, isa := x["name"].(string); isa {
						model = s
					}
				case string:
					model = x
				default:
					panic(x)
				}
			}

			q.regs[i] = v
		}
	}

	return nil
}
