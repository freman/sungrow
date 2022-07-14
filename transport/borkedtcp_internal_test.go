package transport

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBorkedTCPTransporterNormal(t *testing.T) {
	requires := require.New(t)

	server, client := net.Pipe()
	go func() {
		defer server.Close()
		for {
			buf := make([]byte, tcpMaxLength)
			n, err := server.Read(buf)
			requires.NoError(err)
			requires.GreaterOrEqual(n, 7)

			switch buf[7] {
			case 1: // Echo
				n, err = server.Write(buf[:n])
			case 2: // Good exception with correct length (3)
				n, err = server.Write([]byte{0, 1, 0, 0, 0, 3, 1, 0x84, 2})
			case 3: // Bad exception with malformed length (2)
				n, err = server.Write([]byte{0, 1, 0, 0, 0, 2, 1, 0x84, 3})
			}

			requires.NoError(err)
			requires.GreaterOrEqual(n, 7)
		}
	}()

	transporter := &borkedTCPTransport{
		Timeout:     tcpTimeout,
		IdleTimeout: tcpIdleTimeout,
		conn:        client,
	}

	tests := map[string][][]byte{
		"echo":           {[]byte{0, 1, 0, 0, 0, 2, 1, 1}, []byte{0, 1, 0, 0, 0, 2, 1, 1}},
		"good exception": {[]byte{0, 1, 0, 0, 0, 2, 1, 2}, []byte{0, 1, 0, 0, 0, 3, 1, 0x84, 2}},
		"bad exception":  {[]byte{0, 1, 0, 0, 0, 3, 1, 3}, []byte{0, 1, 0, 0, 0, 3, 1, 0x84, 3}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			requires := require.New(t)
			resp, err := transporter.Send(test[0])
			requires.NoError(err)
			requires.Equal(test[1], resp)
		})
	}

	client.Close()
}
