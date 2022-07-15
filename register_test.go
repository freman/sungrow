package sungrow_test

import (
	"testing"

	"github.com/freman/sungrow"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestUnmarshleRegister(t *testing.T) {
	requires := require.New(t)
	sample :=
		`- address: 1
  name: "test1"
  type: "uint32"
- address: 2
  name: "test2"
  type: "uint32"
  scale: 0.01
- address: 3
  name: "test2"
  count: 15`

	var registers []sungrow.Register
	requires.NoError(yaml.Unmarshal([]byte(sample), &registers))

	requires.Equal("uint32", registers[0].Type)
	requires.Equal("uint32", registers[1].Type)
	requires.Equal("int16", registers[2].Type)

	requires.Equal(0, registers[0].Count)
	requires.Equal(0, registers[1].Count)
	requires.Equal(15, registers[2].Count)

	requires.Equal(1.0, registers[0].Scale)
	requires.Equal(0.01, registers[1].Scale)
	requires.Equal(1.0, registers[2].Scale)
}
