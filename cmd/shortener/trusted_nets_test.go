package main

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/serg2014/shortener/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrustedNetsMiddleware(t *testing.T) {
	data := "hello"
	type expect struct {
		StatusCode int
		data       string
	}
	tests := []struct {
		headers       map[string]string
		name          string
		TrustedSubnet config.TrustedSubnet
		expect        expect
	}{
		{
			name:          "1",
			TrustedSubnet: config.TrustedSubnet{},
			expect: expect{
				StatusCode: http.StatusForbidden,
				data:       http.StatusText(http.StatusForbidden) + "\n",
			},
		},
		{
			name: "2",
			headers: map[string]string{
				"X-Real-IP": "not valid",
			},
			TrustedSubnet: config.TrustedSubnet{},
			expect: expect{
				StatusCode: http.StatusForbidden,
				data:       http.StatusText(http.StatusForbidden) + "\n",
			},
		},
		{
			name: "3",
			headers: map[string]string{
				"X-Real-IP": "127.0.0.1",
			},
			TrustedSubnet: config.TrustedSubnet{},
			expect: expect{
				StatusCode: http.StatusForbidden,
				data:       http.StatusText(http.StatusForbidden) + "\n",
			},
		},

		{
			name: "11",
			TrustedSubnet: config.TrustedSubnet{
				Data: &net.IPNet{
					IP:   net.IP([]byte{0x7f, 0x0, 0x0, 0x0}),
					Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0x0}),
				},
			},
			expect: expect{
				StatusCode: http.StatusForbidden,
				data:       http.StatusText(http.StatusForbidden) + "\n",
			},
		},
		{
			name: "21",
			headers: map[string]string{
				"X-Real-IP": "not valid",
			},
			TrustedSubnet: config.TrustedSubnet{
				Data: &net.IPNet{
					IP:   net.IP([]byte{0x7f, 0x0, 0x0, 0x0}),
					Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0x0}),
				},
			},
			expect: expect{
				StatusCode: http.StatusForbidden,
				data:       http.StatusText(http.StatusForbidden) + "\n",
			},
		},
		{
			name: "31",
			headers: map[string]string{
				"X-Real-IP": "192.168.0.1",
			},
			TrustedSubnet: config.TrustedSubnet{
				Data: &net.IPNet{
					IP:   net.IP([]byte{0x7f, 0x0, 0x0, 0x0}),
					Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0x0}),
				},
			},
			expect: expect{
				StatusCode: http.StatusForbidden,
				data:       http.StatusText(http.StatusForbidden) + "\n",
			},
		},
		{
			name: "31",
			headers: map[string]string{
				"X-Real-IP": "127.0.0.1",
			},
			TrustedSubnet: config.TrustedSubnet{
				Data: &net.IPNet{
					IP:   net.IP([]byte{0x7f, 0x0, 0x0, 0x0}),
					Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0x0}),
				},
			},
			expect: expect{
				StatusCode: http.StatusOK,
				data:       data,
			},
		},
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(data))
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tsn := TrustedNetsMiddleware(test.TrustedSubnet)
			// create the handler to test, using our custom "next" handler
			handlerToTest := tsn(nextHandler)
			// create a mock request to use
			req := httptest.NewRequest("GET", "http://localhost/api/internal/stats", nil)
			if len(test.headers) != 0 {
				for k := range test.headers {
					req.Header.Add(k, test.headers[k])
				}
			}

			// call the handler using a mock response recorder (we'll not use that anyway)
			w := httptest.NewRecorder()
			handlerToTest.ServeHTTP(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, test.expect.StatusCode, resp.StatusCode)
			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, test.expect.data, string(respBody))
		})
	}
}
