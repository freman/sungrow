package sungrow

type Registers struct {
	Input   []Register `yaml:"input,omitempty"`
	Holding []Register `yaml:"holding,omitempty"`
}

func (r *Registers) Clear() {
	r.Input = []Register{}
	r.Holding = []Register{}
}
