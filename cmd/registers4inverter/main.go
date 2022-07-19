package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/freman/sungrow"
)

func main() {
	model := flag.String("model", "SH10RT", "Name of the register")
	registers := flag.String("regs", "sungrow.yml", "Register definition file")
	format := flag.String("format", "{{.Address}} {{.Name}}", "Render template")

	flag.Parse()

	*format = strings.ReplaceAll(*format, "\\n", "\n") + "\n"

	template, err := template.New("outut format").Parse(*format)
	if err != nil {
		fmt.Println(err)
		return
	}

	var inv sungrow.Inverter
	if err := inv.DefineFromYaml(*registers); err != nil {
		fmt.Println(err)
		return
	}

	for _, reg := range inv.Registers.Input {
		if reg.Models.ContainsOrNull(*model) {
			template.Execute(os.Stdout, reg)
		}
	}
}
