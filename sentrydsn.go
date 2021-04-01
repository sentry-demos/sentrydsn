// Package sentrydsn provides a utility for deriving client DSN keys from incoming http requests.
package sentrydsn

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

const http_x_sentry_auth = "X-Sentry-Auth"

var (
	// ErrMissing User Thrown if we are missing the public key that comprises {PROTOCOL}://{PUBLIC_KEY}:{SECRET_KEY}@{HOST}{PATH}/{PROJECT_ID}
	ErrMissingUser      = errors.New("sentry:  missing public key")
	ErrMissingProjectID = errors.New("sentry:  Failed attempt to parse project ID from path --")
)
var pk_re = regexp.MustCompile(`sentry_key=([a-f0-9]{32})`)
var sk_re = regexp.MustCompile(`sentry_secret=([a-f0-9]{32})`)
var path_re = regexp.MustCompile(`\/api\/\d+\/store\/`)
var legacy_re = regexp.MustCompile(`\/api\/store\/`)
var envelope_re = regexp.MustCompile(`\/api\/\d+\/envelope\/`)

type DSN struct {
	URL       string //original dsn for incoming request
	Host      string
	ProjectID string
	PublicKey string
	SecretKey string
}
type User struct {
	PublicKey string //public key for DSN
	SecretKey string //private key for DSN if necessary
}

// FromRequest takes a Request struct as a parameter and returns a DSN struct providing the client's DSN via myDSN.URL
// Critical assumption here is that User information (sentry_key and optionally sentry_secret) will come from either
// request headers or the request query string.
// You will never use both to fill each of these values.
// We parse headers first to find User info. This will return pk, sk, both or err if no pk is found.
// If we err using headers we proceed to the QS. An Err here throws for the entire FromRequest operation.
func FromRequest(r *http.Request) (*DSN, error) {

	var user *User
	u := r.URL //represents a fully parsed url
	h := r.Header.Get(http_x_sentry_auth)

	host := u.Hostname()
	if len(host) == 0 {
		host = r.Host
	}
	//some routers/proxies may strip the host from http.Request.URL so http.Request.Host is useful.

	usingHeader, err := parseHeaders(h)
	if err != nil {

		usingQs, qerr := parseQueryString(u)

		if qerr != nil {
			return nil, ErrMissingUser
		} else {
			user = usingQs
		}
	} else {
		user = usingHeader
	}
	// parse project
	p, err := checkPath(u)
	if err != nil {
		return nil, err
	}
	// complete DSN
	dsn := createDSN(user, host, p)

	return dsn, nil

}

// parseHeaders parses values from the X-Sentry-Auth header. Searches for both pk and sk values.
// It throws an error if nothing is found for pk as this is critical
// Returns user struct with appropriate values or empty strings.
func parseHeaders(h string) (*User, error) {

	var sentryPublic string
	var sentrySecret string

	if len(h) == 0 {
		return nil, ErrMissingUser
	}

	toArray := strings.Split(strings.SplitN(h, " ", 2)[1], ",")
	//Anticipates header: Sentry <start-header-values,...>

	for _, v := range toArray {

		foundPublic := pk_re.MatchString(v)
		foundPrivate := sk_re.MatchString(v)
		if foundPublic {
			sentryPublic = strings.Split(v, "=")[1]
		}
		if foundPrivate {
			sentrySecret = strings.Split(v, "=")[1]
		}
	}
	if len(sentryPublic) == 0 {
		return nil, ErrMissingUser

	}
	return &User{PublicKey: sentryPublic, SecretKey: sentrySecret}, nil

}

// createDSN concatenates our DSN components into a client DSN key.
// In the case where we encounter the legacy /api/store/ the returned DSN struct will have url == ""
// This allows for optional checks in case the other parts of the struct (publicKey) are used for projectID lookups
func createDSN(d *User, host string, projectID string) *DSN {

	var url string
	prefix := "https://"
	if len(projectID) == 0 {
		url = ""
	} else if len(d.PublicKey) > 0 && len(d.SecretKey) == 0 {
		url = fmt.Sprintf("%v%v@%v/%v", prefix, d.PublicKey, host, projectID)
	} else if len(d.PublicKey) > 0 && len(d.SecretKey) > 0 {
		url = fmt.Sprintf("%v%v:%v@%v/%v", prefix, d.PublicKey, d.SecretKey, host, projectID)
	}

	return &DSN{URL: url, ProjectID: projectID, Host: host, PublicKey: d.PublicKey, SecretKey: d.SecretKey}
}

// parseQueryString parses sentry public and secret keys from the query string where available.
// Function throws if we are missing pk as this is critical.
// Returns User struct with parsed values or empty strings if value was not available.
func parseQueryString(u *url.URL) (*User, error) {

	pk := u.Query().Get("sentry_key")
	if len(pk) == 0 {
		return nil, ErrMissingUser
	}
	sk := u.Query().Get("sentry_secret")

	return &User{PublicKey: pk, SecretKey: sk}, nil

}

// checkPath validates anticipated path structure /api/<project_id>/store/ OR /api/store/ OR /api/<project_id>/envelope/ and returns a projectID.
// The legacy /api/store/ endpoint does not include project id.
// This edge case is usually where a public key could be used to lookup project meta data
// in Relay. As we are not in relay this is not an option.
// Older clients tested:
//
// raven-python 5.27.0
// java Raven-Java 7.8.0-31c26
// javascript raven-js 3.10.0
// cocoa 4.3.3
//
// All of these clients utilize the  /api/<project_id>/store/  endpoint.
// Given the test we have a higher degree of certainty that we will not encounter the legacy api
// and all incoming requests will have a project id in path.
func checkPath(u *url.URL) (string, error) {

	path := u.Path
	isValid := path_re.MatchString(path)
	isValidLegacy := legacy_re.MatchString(path)
	isValidEnvelope := envelope_re.MatchString(path)

	if !isValid && !isValidEnvelope {
		if isValidLegacy {
			return "", nil
		}
		return "", ErrMissingProjectID
	}
	pathItems := strings.Split(path, "/")

	//with leading + trailing splits array has deterministic length of 5
	return pathItems[2], nil

}
