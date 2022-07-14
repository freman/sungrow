package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/goburrow/modbus"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

const websocketConnectResponse = `{
	"result_code":  1,
	"result_msg":   "success",
	"result_data":  {
			"service":      "connect",
			"token":        "04794176-350b-4a71-b3d9-fcb0a0582920",
			"uid":  1,
			"tips_disable": 1
	}
}`

const websocketDeviceListResponse = `{
    "result_code":  1,
    "result_msg":   "success",
    "result_data":  {
        "service":  "devicelist",
        "list": [{
                "id":   1,
                "dev_id":   1,
                "dev_code": 3587,
                "dev_type": 35,
                "dev_procotol": 2,
                "inv_type": 0,
                "dev_sn":   "A1234567890",
                "dev_name": "SH10RT(COM1-001)",
                "dev_model":    "SH10RT",
                "port_name":    "COM1",
                "phys_addr":    "1",
                "logc_addr":    "1",
                "link_status":  1,
                "init_status":  1,
                "dev_special":  "0",
                "list": []
            }, {
                "id":   2,
                "dev_id":   2,
                "dev_code": 8427,
                "dev_type": 44,
                "dev_procotol": 1,
                "inv_type": 0,
                "dev_sn":   "S1234567890",
                "dev_name": "SBR224(COM1-200)",
                "dev_model":    "SBR224",
                "port_name":    "COM1",
                "phys_addr":    "200",
                "logc_addr":    "2",
                "link_status":  1,
                "init_status":  255,
                "dev_special":  "0",
                "list": []
            }],
        "count":    2
    }
}`

func TestJSONMessage(t *testing.T) {
	requires := require.New(t)

	body := []byte(websocketConnectResponse)

	var connectMessage jsonConnectMessage
	var message jsonMessage
	message.ResultData = &connectMessage

	requires.NoError(json.Unmarshal(body, &message))
	requires.Equal(1, message.ResultCode)
	requires.Equal("success", message.ResultMsg)
	requires.NotNil(message.ResultData)
	requires.Equal(&connectMessage, message.ResultData)
	requires.Equal("connect", connectMessage.Service)
	requires.Equal("04794176-350b-4a71-b3d9-fcb0a0582920", connectMessage.Token)
	requires.Equal(1, connectMessage.UID)
	requires.Equal(1, connectMessage.TipsDisable)
}

func TestGetParamPackager(t *testing.T) {
	requires := require.New(t)

	packager := httpPackager{
		SlaveID: 1,
	}

	adu, err := packager.Encode(&modbus.ProtocolDataUnit{
		FunctionCode: modbus.FuncCodeReadInputRegisters,
		Data: []byte{
			0, 1, 2, 3,
		},
	})

	requires.NoError(err)
	requires.Equal([]byte{0, 1, 0, 1, 2, 3}, adu)

	adu, err = packager.Encode(&modbus.ProtocolDataUnit{
		FunctionCode: modbus.FuncCodeReadHoldingRegisters,
		Data: []byte{
			0, 1, 2, 3,
		},
	})

	requires.NoError(err)
	requires.Equal([]byte{1, 1, 0, 1, 2, 3}, adu)

	adu, err = packager.Encode(&modbus.ProtocolDataUnit{
		FunctionCode: modbus.FuncCodeReadCoils,
		Data: []byte{
			0, 1, 2, 3,
		},
	})

	requires.Error(err)
	requires.Empty(adu)
}

func TestGetParamTransporter(t *testing.T) {
	requires := require.New(t)

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ws/home/overview":
			c, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
			requires.NoError(err)
			defer c.Close()
			for {
				mt, message, err := c.ReadMessage()
				if err != nil {
					return
				}

				requires.Equal(websocket.TextMessage, mt)

				switch {
				case bytes.Equal([]byte(websocketConnectMessage), message):
					requires.NoError(c.WriteMessage(mt, []byte(websocketConnectResponse)))
				case bytes.Equal([]byte(fmt.Sprintf(websocketDeviceListMessageTemplate, "04794176-350b-4a71-b3d9-fcb0a0582920")), message):
					requires.NoError(c.WriteMessage(mt, []byte(websocketDeviceListResponse)))
				default:
					t.Error("unexpected message")
				}
			}
		case "/device/getParam":
			query := r.URL.Query()
			requires.Equal("04794176-350b-4a71-b3d9-fcb0a0582920", query.Get("token"))
			requires.Contains([]string{"0", "1"}, query.Get("param_type"))
			_, err := w.Write([]byte(`{"result_code":1, "result_msg": "success", "result_data": {"param_value": "00 11 22 33 "}}`))
			requires.NoError(err)
		default:
			t.Errorf("Unexpected url %q", r.URL.Path)
		}
	}))
	defer svr.Close()

	host, port, err := net.SplitHostPort(svr.Listener.Addr().String())
	requires.NoError(err)

	iPort, err := strconv.Atoi(port)
	requires.NoError(err)

	transporter := &httpTransporter{
		Host:     host,
		WSPort:   iPort,
		HTTPPort: iPort,
	}

	resp, err := transporter.Send([]byte{0, 1, 0, 1, 2, 3})
	requires.NoError(err)
	requires.Equal([]byte{0, 4, 0, 0x11, 0x22, 0x33}, resp)

	resp, err = transporter.Send([]byte{1, 1, 0, 1, 2, 3})
	requires.NoError(err)
	requires.Equal([]byte{1, 4, 0, 0x11, 0x22, 0x33}, resp)
}
