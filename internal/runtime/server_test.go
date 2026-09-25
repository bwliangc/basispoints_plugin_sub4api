package runtime

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	pluginv1 "example.com/basispoints-transport/pkg/pluginapi/v1"
	"google.golang.org/grpc"
)

func TestAccountConfig(t *testing.T) {
	for _, raw := range []string{"", "{}", `{"enabled":true}`, `{"account_ids":null}`, `{"account_ids":[]}`} {
		cfg, err := normalizeConfig([]byte(raw))
		if err != nil || accountAllowed(12, cfg.AccountIDs) {
			t.Fatalf("empty/legacy config %q must not route accounts: %v", raw, err)
		}
	}
	cfg, err := normalizeConfig([]byte(`{"account_ids":[12,34,12]}`))
	if err != nil || !reflect.DeepEqual(cfg.AccountIDs, []int64{12, 34}) {
		t.Fatalf("normalization failed: %v, %v", cfg.AccountIDs, err)
	}
	for _, raw := range []string{`{"account_ids":[0]}`, `{"account_ids":[-1]}`, `{"account_ids":[1.5]}`, `{"account_ids":["12"]}`, `{"account_ids":[9007199254740992]}`} {
		if _, err := normalizeConfig([]byte(raw)); err == nil {
			t.Errorf("accepted invalid config %s", raw)
		}
	}
}

type testForwardStream struct {
	grpc.ServerStream
	requests  []*pluginv1.ForwardRequest
	responses []*pluginv1.ForwardResponse
}

func (*testForwardStream) Context() context.Context { return context.Background() }
func (s *testForwardStream) Recv() (*pluginv1.ForwardRequest, error) {
	if len(s.requests) == 0 {
		return nil, io.EOF
	}
	r := s.requests[0]
	s.requests = s.requests[1:]
	return r, nil
}
func (s *testForwardStream) Send(r *pluginv1.ForwardResponse) error {
	s.responses = append(s.responses, r)
	return nil
}

func TestForwardAccountRouting(t *testing.T) {
	for _, tc := range []struct {
		name     string
		account  int64
		allowed  []int64
		model    string
		disabled bool
		wantBPS  bool
	}{
		{name: "selected account", account: 12, allowed: []int64{12, 34}, model: "gpt-6-astra", wantBPS: true},
		{name: "second selected account", account: 34, allowed: []int64{12, 34}, model: "gpt-6-astra", wantBPS: true},
		{name: "unselected account", account: 56, allowed: []int64{12, 34}, model: "gpt-6-astra"},
		{name: "empty allowlist", account: 12, model: "gpt-6-astra"},
		{name: "missing host account", allowed: []int64{12}, model: "gpt-6-astra"},
		{name: "invalid host account", account: -1, allowed: []int64{12}, model: "gpt-6-astra"},
		{name: "model mismatch", account: 12, allowed: []int64{12}, model: "other-model"},
		{name: "routing disabled", account: 12, allowed: []int64{12}, model: "gpt-6-astra", disabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte(`{"model":"` + tc.model + `","input":"hello","tools":[{"type":"function","name":"lookup","parameters":{"type":"object"}}]}`)
			type receivedRequest struct{ path, body, marker string }
			received := make(chan receivedRequest, 1)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				received <- receivedRequest{r.URL.RequestURI(), string(raw), r.Header.Get("X-Test-Passthrough")}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"id":"resp_test","output":[]}`)
			}))
			defer upstream.Close()
			s := NewServer()
			s.config.Endpoint = upstream.URL + "/bps"
			s.config.AccountIDs = tc.allowed
			s.config.Enabled = !tc.disabled
			stream := &testForwardStream{requests: []*pluginv1.ForwardRequest{
				{Frame: &pluginv1.ForwardRequest_Start{Start: &pluginv1.ForwardRequestStart{
					Method: "POST", Url: upstream.URL + "/original?keep=1", AccountId: tc.account,
					Headers: httpHeadersToProto(http.Header{
						"Authorization": {"Bearer test-token"}, "Chatgpt-Account-Id": {"12"}, "X-Test-Passthrough": {"preserved"},
					}),
				}}},
				{Frame: &pluginv1.ForwardRequest_BodyChunk{BodyChunk: body}},
				{Frame: &pluginv1.ForwardRequest_BodyEnd{BodyEnd: true}},
			}}
			if err := s.Forward(stream); err != nil {
				t.Fatal(err)
			}
			for _, response := range stream.responses {
				if e := response.GetError(); e != nil {
					t.Fatalf("forward error: %s", e.GetMessage())
				}
			}
			select {
			case got := <-received:
				if tc.wantBPS {
					if got.path != "/bps" || got.body == string(body) {
						t.Fatalf("expected transformed BasisPoints request, got %+v", got)
					}
				} else if got.path != "/original?keep=1" || got.body != string(body) || got.marker != "preserved" {
					t.Fatalf("passthrough request changed: %+v", got)
				}
			default:
				t.Fatal("no upstream request received")
			}
		})
	}
}
