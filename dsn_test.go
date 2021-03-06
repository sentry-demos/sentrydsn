package sentrydsn

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

//setup

type testRequest struct {
	url         string
	header      []string
	body        map[string]string
	description string
	expected    string
}

var testTableLegacyUserInfo = []testRequest{
	// sentry_secret & sentry_key in query string
	{"https://sentry.io/api/1234/store/?sentry_secret=4784fbc50de2473f9977cfce8a9adce5&sentry_key=4784fbc50de2473f9977cfce8a9adce5&sentry_version=7",
		[]string{"Sentry sentry_version=7", "sentry_client=<client>", "sentry_timestamp=1614144877.269"},

		map[string]string{
			"name": "testbody",
		}, "Testing user info in querystring",
		"https://4784fbc50de2473f9977cfce8a9adce5:4784fbc50de2473f9977cfce8a9adce5@sentry.io/1234"},
	//sentry_secret & sentry_key in X-SENTRY-AUTH header
	{"https://o87286.ingest.sentry.io/api/1234/store/?&sentry_version=7",
		[]string{"Sentry sentry_version=7", "sentry_client=<client>", "sentry_timestamp=1614144877.269", "sentry_secret=4784fbc50de2473f9977cfce8a9adce5", "sentry_key=4784fbc50de2473f9977cfce8a9adce5"},
		map[string]string{
			"name": "testbody",
		}, "Testing user info in headers",
		"https://4784fbc50de2473f9977cfce8a9adce5:4784fbc50de2473f9977cfce8a9adce5@o87286.ingest.sentry.io/1234"},
}
var testTableMissingUserInfo = []testRequest{
	//ONLY sentry_secret in query string
	{"https://sentry.io/api/1234/store/?sentry_secret=4784fbc50de2473f9977cfce8a9adce5&sentry_version=7",
		[]string{"Sentry sentry_version=7", "sentry_client=<client>", "sentry_timestamp=1614144877.269"},
		map[string]string{
			"name": "testbody",
		}, "ONLY sentry_secret in query string",
		""},
	//ONLY sentry_secret in X-SENTRY-AUTH header
	{"https://sentry.io/api/1234/store/?&sentry_version=7",
		[]string{"Sentry sentry_version=7", "sentry_client=<client>", "sentry_timestamp=1614144877.269", "sentry_secret=4784fbc50de2473f9977cfce8a9adce5"},
		map[string]string{
			"name": "testbody",
		}, "ONLY sentry_secret in X-SENTRY-AUTH header",
		""},
}

var testTableProjectID = []testRequest{
	// sentry_secret & sentry_key in query string
	{"https://sentry.io/apistore/?sentry_secret=4784fbc50de2473f9977cfce8a9adce5&sentry_key=4784fbc50de2473f9977cfce8a9adce5&sentry_version=7",
		[]string{"Sentry sentry_version=7", "sentry_client=<client>", "sentry_timestamp=1614144877.269"},
		map[string]string{
			"name": "testbody",
		}, "Testing malformed api path",
		"https://4784fbc50de2473f9977cfce8a9adce5:4784fbc50de2473f9977cfce8a9adce5@sentry.io"},
	//sentry_secret & sentry_key in X-SENTRY-AUTH header
	{"https://sentry.io//api//1234///store//?&sentry_version=7",
		[]string{"Sentry sentry_version=7", "sentry_client=<client>", "sentry_timestamp=1614144877.269", "sentry_secret=4784fbc50de2473f9977cfce8a9adce5", "sentry_key=4784fbc50de2473f9977cfce8a9adce5"},
		map[string]string{
			"name": "testbody",
		}, "Testing malformed api path",
		"https://4784fbc50de2473f9977cfce8a9adce5:4784fbc50de2473f9977cfce8a9adce5@sentry.io/1234"},
}
var testTableLegacy = []testRequest{
	// sentry_secret & sentry_key in query string
	{"https://sentry.io/api/store/?sentry_secret=4784fbc50de2473f9977cfce8a9adce5&sentry_key=4784fbc50de2473f9977cfce8a9adce5&sentry_version=7",
		[]string{"Sentry sentry_version=7", "sentry_client=<client>", "sentry_timestamp=1614144877.269"},
		map[string]string{
			"name": "testbody",
		}, "Testing legacy api path",
		"len(dsn.URL) == 0"},
}

var testTableEnvelopeEndpoint = []testRequest{
	//ONLY sentry_secret in query string
	{"https://sentry.io/api/1234/envelope/?sentry_secret=4784fbc50de2473f9977cfce8a9adce5&sentry_key=4784fbc50de2473f9977cfce8a9adce5&sentry_version=7",
		[]string{"Sentry sentry_version=7", "sentry_client=<client>", "sentry_timestamp=1614144877.269"},
		map[string]string{
			"name": "testbody",
		}, "ONLY sentry_secret in query string",
		"https://4784fbc50de2473f9977cfce8a9adce5:4784fbc50de2473f9977cfce8a9adce5@sentry.io/1234"},
	//ONLY sentry_secret in X-SENTRY-AUTH header
	{"https://sentry.io/api/1234/envelope/?&sentry_version=7",
		[]string{"Sentry sentry_version=7", "sentry_client=<client>", "sentry_timestamp=1614144877.269", "sentry_secret=4784fbc50de2473f9977cfce8a9adce5"},
		map[string]string{
			"name": "testbody",
		}, "ONLY sentry_secret in X-SENTRY-AUTH header",
		""},
}

//tests

func TestLegacyUserRequest(t *testing.T) {
	for _, test := range testTableLegacyUserInfo {
		rb, _ := json.Marshal(test.body)
		r := httptest.NewRequest("POST", test.url, bytes.NewBuffer(rb))
		r.Header.Set("X-SENTRY-AUTH", strings.Join(test.header, ", "))
		got, _ := FromRequest(r)

		//check that legacy DSN is correct
		if got.URL != test.expected {
			t.Errorf("Expected -- %s -- Got %s", test.expected, got.URL)
		}

	}
}

func TestMissingPublicKey(t *testing.T) {
	//test needed to validate errors for missing public keys
	for _, test := range testTableMissingUserInfo {
		rb, _ := json.Marshal(test.body)
		r := httptest.NewRequest("POST", test.url, bytes.NewBuffer(rb))
		r.Header.Set("X-SENTRY-AUTH", strings.Join(test.header, ", "))
		got, _ := FromRequest(r)
		if got != nil {
			t.Errorf("Expected -- %s -- Got %s", ErrMissingUser, got)
		}
	}

}

func TestMissingProjectID(t *testing.T) {
	//test needed to validate errors for missing project IDs
	//test includes check for legacy store endpoint api/store. This should return a dsn without projectID.
	for _, test := range testTableProjectID {
		rb, _ := json.Marshal(test.body)
		r := httptest.NewRequest("POST", test.url, bytes.NewBuffer(rb))
		r.Header.Set("X-SENTRY-AUTH", strings.Join(test.header, ", "))
		got, err := FromRequest(r)
		if got != nil {
			if got.URL != test.expected {
				t.Errorf("Expected -- %s -- Got %s", test.expected, got.URL)
			}

		} else if err != ErrMissingProjectID {
			t.Errorf("Expected -- %s -- Got %s", ErrMissingProjectID, err)
		}
	}
}

func TestLegacyStoreAPI(t *testing.T) {
	for _, test := range testTableLegacy {
		rb, _ := json.Marshal(test.body)
		r := httptest.NewRequest("POST", test.url, bytes.NewBuffer(rb))
		r.Header.Set("X-SENTRY-AUTH", strings.Join(test.header, ", "))
		got, err := FromRequest(r)
		if err != nil {
			t.Errorf("Expected -- %s -- Got %s", test.expected, err)

		} else if len(got.URL) != 0 {
			t.Errorf("Expected -- %s -- Got %d", test.expected, len(got.URL))

		}
	}
}

func TestEnvelopeEndpoint(t *testing.T) {
	for _, test := range testTableEnvelopeEndpoint {
		rb, _ := json.Marshal(test.body)
		r := httptest.NewRequest("POST", test.url, bytes.NewBuffer(rb))
		r.Header.Set("X-SENTRY-AUTH", strings.Join(test.header, ", "))
		got, err := FromRequest(r)
		if err != nil {
			if err != ErrMissingUser {
				t.Errorf("Expected -- %s -- Got %s", ErrMissingUser, err)
			}
		} else if got.URL != test.expected {
			t.Errorf("Expected -- %s -- Got %s", test.expected, got.URL)
		}
	}
}
