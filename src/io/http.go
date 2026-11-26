package io

// The parts of dart:io's HTTP client (http.dart) that painting's
// NetworkImage uses: HttpClient.getUrl/autoUncompress,
// HttpClientRequest.headers/close, HttpHeaders.add, HttpStatus.
// NewHttpClient implements them over net/http.

import (
	"bytes"
	gonet "net/http"
	"net/url"

	"github.com/jonathanperis/flutter-go/src/async"
)

// HttpStatus codes Flutter's API mentions (dart:io's HttpStatus).
const (
	HttpStatusOk       = 200
	HttpStatusNotFound = 404
)

// HttpClient is an HTTP client (the members Flutter uses).
type HttpClient interface {
	// GetUrl opens a GET request to u.
	GetUrl(u *url.URL) async.Future[HttpClientRequest]
	// SetAutoUncompress sets whether responses are decompressed
	// (autoUncompress=).
	SetAutoUncompress(v bool)
}

// HttpClientRequest is an open request.
type HttpClientRequest interface {
	Headers() HttpHeaders
	// Close sends the request and returns the response.
	Close() async.Future[HttpClientResponse]
}

// HttpHeaders are a request's headers.
type HttpHeaders interface {
	Add(name string, value any)
}

// NewHttpClient is HttpClient(): an HttpClient over net/http. Requests run
// on goroutines and complete through the current event loop's posted
// events (async.Loop.RunPosted / WaitPosted).
func NewHttpClient() HttpClient { return &netClient{autoUncompress: true} }

type netClient struct{ autoUncompress bool }

func (c *netClient) SetAutoUncompress(v bool) { c.autoUncompress = v }

func (c *netClient) GetUrl(u *url.URL) async.Future[HttpClientRequest] {
	return async.FutureValue[HttpClientRequest](&netRequest{client: c, url: u, headers: netHeaders{}})
}

type netHeaders map[string][]string

func (h netHeaders) Add(name string, value any) {
	s, _ := value.(string)
	h[name] = append(h[name], s)
}

type netRequest struct {
	client  *netClient
	url     *url.URL
	headers netHeaders
}

func (r *netRequest) Headers() HttpHeaders { return r.headers }

func (r *netRequest) Close() async.Future[HttpClientResponse] {
	loop := async.Current()
	completer := async.NewCompleter[HttpClientResponse]()
	go func() {
		req, err := gonet.NewRequest("GET", r.url.String(), nil)
		var body []byte
		var resp *gonet.Response
		if err == nil {
			for k, vs := range r.headers {
				for _, v := range vs {
					req.Header.Add(k, v)
				}
			}
			tr := &gonet.Transport{DisableCompression: !r.client.autoUncompress}
			resp, err = (&gonet.Client{Transport: tr}).Do(req)
			if err == nil {
				var buf bytes.Buffer
				_, err = buf.ReadFrom(resp.Body)
				resp.Body.Close()
				body = buf.Bytes()
			}
		}
		loop.Post(func() {
			if err != nil {
				completer.CompleteError(err)
				return
			}
			completer.Complete(&netResponse{status: resp.StatusCode, body: body, length: int(resp.ContentLength)})
		})
	}()
	return completer.Future()
}

type netResponse struct {
	status int
	body   []byte
	length int
}

func (r *netResponse) StatusCode() int    { return r.status }
func (r *netResponse) ContentLength() int { return r.length }
func (r *netResponse) CompressionState() HttpClientResponseCompressionState {
	return HttpClientResponseCompressionStateNotCompressed
}
func (r *netResponse) Listen(onData func([]byte), opts ...async.StreamListenOptions) async.StreamSubscription[[]byte] {
	return async.StreamFromIterable(func(yield func([]byte) bool) { yield(r.body) }).Listen(onData, opts...)
}
