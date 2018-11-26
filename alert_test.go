// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

var (
	testAlert = Alert{
		CID:                "/alert/1234",
		AcknowledgementCID: &[]string{"/acknowledgement/1234"}[0],
		AlertURL:           "https://example.circonus.com/fault-detection?alert_id=1234",
		BrokerCID:          "/broker/1234",
		CheckCID:           "/check/1234",
		CheckName:          "foo bar",
		ClearedOn:          &[]uint{1483033602}[0],
		ClearedValue:       &[]string{"1234"}[0],
		Maintenance:        []string{},
		MetricLinkURL:      &[]string{"http://example.com/docs/what_to_do_when/foo_bar_failure.html"}[0],
		MetricName:         "baz",
		MetricNotes:        &[]string{"blah blah blah"}[0],
		OccurredOn:         1483033102,
		RuleSetCID:         "/rule_set/1234_baz",
		Severity:           2,
		Tags:               []string{"cat:tag"},
		Value:              "5678",
	}
)

func testAlertServer() *httptest.Server {
	f := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/alert/1234" {
			switch r.Method {
			case "GET":
				ret, err := json.Marshal(testAlert)
				if err != nil {
					panic(err)
				}
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, string(ret))
			default:
				w.WriteHeader(404)
				fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
			}
		} else if path == "/alert" {
			switch r.Method {
			case "GET":
				reqURL := r.URL.String()
				var c []Alert
				if reqURL == "/alert?search=%28host%3D%22somehost.example.com%22%29" {
					c = []Alert{testAlert}
				} else if reqURL == "/alert?f__cleared_on=null" {
					c = []Alert{testAlert}
				} else if reqURL == "/alert?f__cleared_on=null&search=%28host%3D%22somehost.example.com%22%29" {
					c = []Alert{testAlert}
				} else if reqURL == "/alert" {
					c = []Alert{testAlert}
				} else {
					c = []Alert{}
				}
				if len(c) > 0 {
					ret, err := json.Marshal(c)
					if err != nil {
						panic(err)
					}
					w.WriteHeader(200)
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprintln(w, string(ret))
				} else {
					w.WriteHeader(404)
					fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, reqURL))
				}
			default:
				w.WriteHeader(404)
				fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
			}
		} else {
			w.WriteHeader(404)
			fmt.Fprintln(w, fmt.Sprintf("not found: %s %s", r.Method, path))
		}
	}

	return httptest.NewServer(http.HandlerFunc(f))
}

func TestNewAlert(t *testing.T) {
	bundle := NewAlert()
	actualType := reflect.TypeOf(bundle)
	expectedType := "*apiclient.Alert"
	if actualType.String() != expectedType {
		t.Fatalf("Expected %s, got %s", expectedType, actualType.String())
	}
}

func TestFetchAlert(t *testing.T) {
	server := testAlertServer()
	defer server.Close()

	ac := &Config{
		TokenKey: "abc123",
		TokenApp: "test",
		URL:      server.URL,
	}
	apih, err := NewAPI(ac)
	if err != nil {
		t.Errorf("Expected no error, got '%v'", err)
	}

	tests := []struct {
		id           string
		cid          string
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"empty cid", "", "", true, "invalid alert CID (none)"},
		{"invalid cid", "/invalid", "", true, "invalid alert CID (/alert//invalid)"},
		{"short cid", "1234", "*apiclient.Alert", false, ""},
		{"long cid", "/alert/1234", "*apiclient.Alert", false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			alert, err := apih.FetchAlert(CIDType(&test.cid))
			if test.shouldFail {
				if err == nil {
					t.Fatal("expected error")
				} else if err.Error() != test.expectedErr {
					t.Fatalf("unexpected error (%s)", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error (%s)", err)
				} else if reflect.TypeOf(alert).String() != test.expectedType {
					t.Fatalf("unexpected type (%s)", reflect.TypeOf(alert).String())
				}
			}
		})
	}
}

func TestFetchAlerts(t *testing.T) {
	server := testAlertServer()
	defer server.Close()

	ac := &Config{
		TokenKey: "abc123",
		TokenApp: "test",
		URL:      server.URL,
	}
	apih, err := NewAPI(ac)
	if err != nil {
		t.Errorf("Expected no error, got '%v'", err)
	}

	alerts, err := apih.FetchAlerts()
	if err != nil {
		t.Fatalf("Expected no error, got '%v'", err)
	}

	actualType := reflect.TypeOf(alerts)
	expectedType := "*[]apiclient.Alert"
	if actualType.String() != expectedType {
		t.Fatalf("Expected %s, got %s", expectedType, actualType.String())
	}

}

func TestSearchAlerts(t *testing.T) {
	server := testAlertServer()
	defer server.Close()

	var apih *API

	ac := &Config{
		TokenKey: "abc123",
		TokenApp: "test",
		URL:      server.URL,
	}
	apih, err := NewAPI(ac)
	if err != nil {
		t.Errorf("Expected no error, got '%v'", err)
	}

	expectedType := "*[]apiclient.Alert"
	search := SearchQueryType(`(host="somehost.example.com")`)
	filter := SearchFilterType(map[string][]string{"f__cleared_on": {"null"}})

	tests := []struct {
		id           string
		search       *SearchQueryType
		filter       *SearchFilterType
		expectedType string
		shouldFail   bool
		expectedErr  string
	}{
		{"no search, no filter", nil, nil, expectedType, false, ""},
		{"search no filter", &search, nil, expectedType, false, ""},
		{"filter no search", nil, &filter, expectedType, false, ""},
		{"both filter and search", &search, &filter, expectedType, false, ""},
	}

	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			ack, err := apih.SearchAlerts(test.search, test.filter)
			if test.shouldFail {
				if err == nil {
					t.Fatal("expected error")
				} else if err.Error() != test.expectedErr {
					t.Fatalf("unexpected error (%s)", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error (%s)", err)
				} else if reflect.TypeOf(ack).String() != test.expectedType {
					t.Fatalf("unexpected type (%s)", reflect.TypeOf(ack).String())
				}
			}
		})
	}
}
