package appium

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type reconnectTransport func(*http.Request) (*http.Response, error)

func (f reconnectTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestReconnectOnlyRetriesReadTransportFailure(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: reconnectTransport(func(*http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return nil, errors.New("connection reset")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"value":"observed"}`)), Header: make(http.Header)}, nil
			})}
			d := &Device{cfg: Config{URL: "http://appium.invalid"}, http: client}
			var value string
			err := d.call(method, "/session/test/source", nil, &value)
			if method == http.MethodGet {
				if err != nil || calls != 2 || value != "observed" {
					t.Fatalf("read reconnect: calls=%d value=%q err=%v", calls, value, err)
				}
			} else {
				var deviceErr DeviceError
				if !errors.As(err, &deviceErr) || calls != 1 {
					t.Fatalf("mutation replayed or failure lost: calls=%d err=%v", calls, err)
				}
			}
		})
	}
}

func TestReconnectIsBoundedAndDoesNotRetryProtocolErrors(t *testing.T) {
	for _, protocol := range []bool{false, true} {
		calls := 0
		d := &Device{cfg: Config{URL: "http://appium.invalid"}, http: &http.Client{Transport: reconnectTransport(func(*http.Request) (*http.Response, error) {
			calls++
			if !protocol {
				return nil, errors.New("offline")
			}
			return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader(`{"value":{"error":"invalid session id","message":"gone"}}`)), Header: make(http.Header)}, nil
		})}}
		err := d.call(http.MethodGet, "/session/test/source", nil, nil)
		var deviceErr DeviceError
		want := 2
		if protocol {
			want = 1
		}
		if !errors.As(err, &deviceErr) || calls != want {
			t.Fatalf("protocol=%v calls=%d err=%v", protocol, calls, err)
		}
	}
}

func TestReconnectKeepsResponseLimit(t *testing.T) {
	calls := 0
	d := &Device{cfg: Config{URL: "http://appium.invalid"}, http: &http.Client{Transport: reconnectTransport(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("reset")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"value":"too big"}`)), Header: make(http.Header)}, nil
	})}}
	err := d.callLimited(http.MethodGet, "/source", nil, nil, 5)
	if err == nil || err.Error() != "Appium response exceeds artifact size limit" || calls != 2 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestReconnectRedactsSecretTransportFailures(t *testing.T) {
	calls := 0
	d := &Device{cfg: Config{URL: "http://appium.invalid", SecretFields: map[string]string{"Password": "SECRET"}}, http: &http.Client{Transport: reconnectTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("transport echoed secret-value")
	})}}
	err := d.call(http.MethodGet, "/source", nil, nil)
	if err == nil || err.Error() != "Appium request failed during secret-enabled run" || calls != 2 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
