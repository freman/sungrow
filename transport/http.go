package transport

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goburrow/modbus"
	"github.com/gorilla/websocket"
)

const (
	// Modbus Application Protocol
	jsonMaxLength = 4096

	// Default HTTp timeout is not set
	httpTimeout = 10 * time.Second
	httpPort    = 80

	// Custom param types because why not
	paramTypeInputRegister   byte = 0
	paramTypeHoldingRegister byte = 1

	// Websocket specific data
	websocketPort                      = 8082
	websocketConnectMessage            = `{"lang":"en_us", "token":"", "service":"connect"}`
	websocketDeviceListMessageTemplate = `{"lang":"en_us", "token":"%s", "service":"devicelist", "type":"0", "is_check_token":"0" }))`
)

// HTTPClientHandler implements Packager and Transporter interface.
type HTTPClientHandler struct {
	httpPackager
	httpTransporter
}

// NewTCPClientHandler allocates a new TCPClientHandler.
func NewHTTPClientHandler(host string) *HTTPClientHandler {
	h := &HTTPClientHandler{}
	h.Host = host
	h.WSPort = websocketPort
	h.HTTPPort = httpPort
	h.Timeout = httpTimeout
	return h
}

// httpPackager implements Packager interface.
type httpPackager struct {
	// Broadcast address is 0
	SlaveID byte
}

// Encode converts the PDU to binary
func (mb *httpPackager) Encode(pdu *modbus.ProtocolDataUnit) (adu []byte, err error) {
	ok := pdu.FunctionCode == modbus.FuncCodeReadInputRegisters
	ok = ok || pdu.FunctionCode == modbus.FuncCodeReadHoldingRegisters

	if !ok {
		return adu, errors.New("not yet supported")
	}

	paramType := paramTypeInputRegister
	if pdu.FunctionCode == modbus.FuncCodeReadHoldingRegisters {
		paramType = paramTypeHoldingRegister
	}

	// Square peg round hole.
	return append([]byte{
		paramType,
		mb.SlaveID,
	}, pdu.Data...), nil
}

// Verify does nothing....
func (mb *httpPackager) Verify(aduRequest []byte, aduResponse []byte) (err error) {
	return nil
}

// Decode extracts PDU from JSON blob:
func (mb *httpPackager) Decode(adu []byte) (pdu *modbus.ProtocolDataUnit, err error) {
	if len(adu) < 2 {
		return nil, errors.New("invalid data length")
	}

	// Extract the unnessicarily packed data
	pdu = &modbus.ProtocolDataUnit{
		FunctionCode: modbus.FuncCodeReadInputRegisters,
		Data:         adu[1:],
	}

	if adu[0] == paramTypeHoldingRegister {
		pdu.FunctionCode = modbus.FuncCodeReadHoldingRegisters
	}

	return
}

// httpTransporter implements Transporter interface.
type httpTransporter struct {
	// Connect string
	Host     string
	HTTPPort int
	WSPort   int

	// Connect & Read timeout
	Timeout time.Duration
	// Transmission logger
	Logger *log.Logger

	// TCP connection
	mu sync.Mutex

	token   string
	devType int
	devCode int

	getParamURL  *url.URL
	websocketURL *url.URL
}

// Send sends data to server and ensures response length is greater than header length.
func (mb *httpTransporter) Send(aduRequest []byte) (aduResponse []byte, err error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	// Establish a new connection if not connected
	if err = mb.connect(); err != nil {
		return
	}

	// Extract the unnessicarily packed data
	paramType := aduRequest[0]
	slaveID := aduRequest[1]
	address := binary.BigEndian.Uint16(aduRequest[2:4])
	quantity := binary.BigEndian.Uint16(aduRequest[4:6])

	vars := url.Values{
		"token":      {mb.token},
		"lang":       {"en_us"},
		"time123456": {strconv.FormatInt(time.Now().UTC().UnixNano()/1e6, 10)},
		"dev_id":     {strconv.Itoa(int(slaveID))},
		"dev_type":   {strconv.Itoa(mb.devType)},
		"dev_code":   {strconv.Itoa(mb.devCode)},
		"type":       {"3"},
		"param_addr": {strconv.Itoa(int(address + 1))},
		"param_num":  {strconv.Itoa(int(quantity))},
		"param_type": {strconv.Itoa(int(paramType))},
	}

	uri := mb.getParamURL
	uri.RawQuery = vars.Encode()

	mb.logf("modbus: sent %q\n", uri.RawQuery)

	// Send data
	c := &http.Client{
		Timeout: mb.Timeout,
	}
	resp, err := c.Get(uri.String())

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var message jsonMessage
	var simpleMessage jsonSimpleMessage
	message.ResultData = &simpleMessage

	if err := json.NewDecoder(io.LimitReader(resp.Body, jsonMaxLength)).Decode(&message); err != nil {
		return nil, fmt.Errorf("failed to decode json response: %w", err)
	}

	mb.logf("modbus: received %v\n", message)

	if message.ResultCode != 1 {
		return nil, fmt.Errorf("%s  (%d)", message.ResultMsg, message.ResultCode)
	}

	aduResponse, err = hex.DecodeString(
		strings.Join(
			strings.Split(
				strings.TrimSpace(simpleMessage.ParamValue),
				" ",
			),
			""),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to parse hex string in response %q: %w", simpleMessage.ParamValue, err)
	}

	aduResponse = append([]byte{paramType, byte(len(aduResponse))}, aduResponse...)

	mb.logf("modbus: transcoded % 0x\n", aduResponse)

	return aduResponse, nil
}

// Connect establishes a new connection to the address in Address.
// Connect and Close are exported so that multiple requests can be done with one session
func (mb *httpTransporter) Connect() error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	return mb.connect()
}

func (mb *httpTransporter) connect() error {
	if mb.getParamURL == nil {
		mb.getParamURL = &url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort(mb.Host, strconv.Itoa(mb.HTTPPort)),
			Path:   "/device/getParam",
		}
	}

	if mb.websocketURL == nil {
		mb.websocketURL = &url.URL{
			Scheme: "ws",
			Host:   net.JoinHostPort(mb.Host, strconv.Itoa(mb.WSPort)),
			Path:   "/ws/home/overview",
		}
	}

	if mb.token == "" {
		c, _, err := websocket.DefaultDialer.Dial(mb.websocketURL.String(), nil)
		if err != nil {
			log.Fatal("dial:", err)
		}

		if err != nil {
			return fmt.Errorf("failed to connect to websocket to get token: %w", err)
		}

		defer c.Close()

		if err := c.WriteMessage(websocket.TextMessage, []byte(websocketConnectMessage)); err != nil {
			return fmt.Errorf("failed to send websocket connect message: %w", err)
		}

		var connectMessage jsonConnectMessage
		var message jsonMessage
		message.ResultData = &connectMessage

		if err := c.ReadJSON(&message); err != nil {
			return fmt.Errorf("failed to retrieve connect message from websocket: %w", err)
		}

		if message.ResultCode != 1 {
			return fmt.Errorf("unexpected response from websocket: %s (%d)", message.ResultMsg, message.ResultCode)
		}

		mb.token = connectMessage.Token
		if mb.token == "" {
			return errors.New("failed to find token in connect message from websocket")
		}

		if err := c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(websocketDeviceListMessageTemplate, mb.token))); err != nil {
			return fmt.Errorf("failed to send device list request to websocket: %w", err)
		}

		var deviceListMessage jsonDeviceListMessage
		message.ResultData = &deviceListMessage
		if err := c.ReadJSON(&message); err != nil {
			return fmt.Errorf("failed to retrieve device list from websocket: %w", err)
		}

		mb.devCode = deviceListMessage.List[0].DevCode
		mb.devType = deviceListMessage.List[0].DevType
	}

	return nil
}

// Close the current connection.
func (mb *httpTransporter) Close() error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	return mb.close()
}

func (mb *httpTransporter) logf(format string, v ...interface{}) {
	if mb.Logger != nil {
		mb.Logger.Printf(format, v...)
	}
}

func (mb *httpTransporter) close() (err error) {
	mb.token = ""
	return
}
