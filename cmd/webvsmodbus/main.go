package main

import (
	"bytes"
	"flag"
	"fmt"

	"github.com/TwiN/go-color"
	"github.com/freman/sungrow"
	"github.com/freman/sungrow/transport"
	"github.com/goburrow/modbus"
)

func main() {
	addr := flag.String("addr", "", "Address of your inverter, eg: 10.0.0.84")
	httpPort := flag.Int("httpPort", 80, "Port of the regular http server")
	wsPort := flag.Int("wsPort", 8082, "Port of the websocket server")
	slaveID := flag.Int("slaveID", 1, "Slave ID")

	inverter := flag.String("inverter", "sungrow.yml", "Inverter definition file")

	flag.Parse()

	if *addr == "" {
		fmt.Println("Hey, you forgot to tell me what to talk to")
		flag.PrintDefaults()
		return
	}

	tcpInv, err := tcpInverter(*addr, *inverter, *slaveID)
	if err != nil {
		fmt.Println("Failed to query tcp modbus", err)
		return
	}

	httpInv, err := httpInverter(*addr, *inverter, *httpPort, *wsPort, *slaveID)
	if err != nil {
		fmt.Println("Failed to query http modbus", err)
		return
	}

	for set, tcpRegs := range [][]sungrow.Register{tcpInv.Registers.Input, tcpInv.Registers.Holding} {
		setName := "input"
		httpRegs := httpInv.Registers.Input

		if set == 1 {
			setName = "holding"
			httpRegs = httpInv.Registers.Holding
		}

		fmt.Println(color.Ize(color.Bold+color.Blue, setName))
		for idx, tcpReg := range tcpRegs {
			output := fmt.Sprintf("\t%s (% 5d+%d):", tcpReg.Name, tcpReg.Address, tcpReg.Count)

			if !tcpReg.Supported {
				fmt.Println(color.Ize(color.Gray, output+" skip"))
				continue
			}

			httpReg := httpRegs[idx]
			if bytes.Equal(httpReg.RAW, tcpReg.RAW) {
				fmt.Println(color.Ize(color.Green, output+" match"))
				continue
			}

			fmt.Printf(color.Ize(color.Red, "%s %v != %v (% 0x != % 0x)\n"), output, httpReg.Value, tcpReg.Value, httpReg.RAW, tcpReg.RAW)

		}
	}
}

func tcpInverter(addr, yamlFile string, slaveID int) (*sungrow.Inverter, error) {
	handler := transport.NewBorkedTCPClient(addr + ":502")
	handler.SlaveID = byte(slaveID)

	//handler.Logger = log.Default()

	client := modbus.NewClient(handler)

	var inv sungrow.Inverter
	if err := inv.DefineFromYaml(yamlFile); err != nil {
		return nil, err
	}

	return &inv, inv.Read(client)
}

func httpInverter(addr, yamlFile string, httpPort, wsPort, slaveID int) (*sungrow.Inverter, error) {
	handler := transport.NewHTTPClientHandler(addr)
	handler.SlaveID = byte(slaveID)
	handler.HTTPPort = httpPort
	handler.WSPort = wsPort

	//	handler.Logger = log.Default()

	client := modbus.NewClient(handler)

	var inv sungrow.Inverter
	if err := inv.DefineFromYaml(yamlFile); err != nil {
		return nil, err
	}

	return &inv, inv.Read(client)
}
