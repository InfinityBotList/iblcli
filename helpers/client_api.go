package helpers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

const (
	apiUrl = "https://spider.infinitybots.gg"
)

// Makes a request to the API
func request(method string, path string, jsonP any, headers map[string]string) (*ClientResponse, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	var body []byte
	var err error
	if jsonP != nil {
		body, err = json.Marshal(jsonP)

		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, apiUrl+path, bytes.NewReader(body))

	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	return &ClientResponse{
		Request:  req,
		Response: resp,
	}, nil
}

// A response from the API
type ClientResponse struct {
	Request  *http.Request
	Response *http.Response
}

// Returns true if the response is a maintenance response (502, 503, 408)
func (c ClientResponse) IsMaint() bool {
	return c.Response.StatusCode == 502 || c.Response.StatusCode == 503 || c.Response.StatusCode == 408
}

// Unmarshals the response body into the given struct
func (c ClientResponse) Json(v any) error {
	return json.NewDecoder(c.Response.Body).Decode(v)
}

// Returns the retry after header. Is a string
func (c ClientResponse) RetryAfter() string {
	return c.Response.Header.Get("Retry-After")
}

// A request to the API
type ClientRequest struct {
	method  string
	path    string
	body    any
	headers map[string]string
}

// Creates a new request
func NewReq() ClientRequest {
	return ClientRequest{
		headers: make(map[string]string),
	}
}

// Sets the method of the request
func (r ClientRequest) Method(method string) ClientRequest {
	r.method = method
	return r
}

// Sets the method to HEAD
func (r ClientRequest) Head(path string) ClientRequest {
	r.method = "HEAD"
	r.path = path
	return r
}

// Sets the method to GET
func (r ClientRequest) Get(path string) ClientRequest {
	r.method = "GET"
	r.path = path
	return r
}

// Sets the method to POST
func (r ClientRequest) Post(path string) ClientRequest {
	r.method = "POST"
	r.path = path
	return r
}

// Sets the method to PUT
func (r ClientRequest) Put(path string) ClientRequest {
	r.method = "PUT"
	r.path = path
	return r
}

// Sets the method to PATCH
func (r ClientRequest) Patch(path string) ClientRequest {
	r.method = "PATCH"
	r.path = path
	return r
}

// Sets the method to DELETE
func (r ClientRequest) Delete(path string) ClientRequest {
	r.method = "DELETE"
	r.path = path
	return r
}

// Sets the path of the request
func (r ClientRequest) Path(path string) ClientRequest {
	r.path = path
	return r
}

// Sets the body of the request
func (r ClientRequest) Json(json any) ClientRequest {
	r.body = json
	return r
}

// Sets the authorization header
func (r ClientRequest) Auth(token string) ClientRequest {
	r.headers["Authorization"] = token
	return r
}

// Sets a header
func (r ClientRequest) Header(key string, value string) ClientRequest {
	r.headers[key] = value
	return r
}

// Executes the request
func (r ClientRequest) Do() (*ClientResponse, error) {
	return request(r.method, r.path, r.body, r.headers)
}
