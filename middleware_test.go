package describer_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"

	"github.com/ar3s3ru/describer"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
)

var testServer *httptest.Server

func printer(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(r.RequestURI))
}

// HTTP server used in TestselfDescribe
func server(port int) http.Handler {
	r := chi.NewRouter()
	r.Use(describer.Middleware())

	r.Get("/get", printer)
	r.Post("/post", printer)
	r.Put("/put", printer)
	r.Patch("/patch", printer)
	r.Delete("/delete", printer)

	r.Route("/route", func(r chi.Router) {
		r.Get("/get", printer)
		r.Post("/post", printer)
		r.Put("/put", printer)
		r.Patch("/patch", printer)
		r.Delete("/delete", printer)
		r.Route("/test", func(r chi.Router) {
			r.Get("/hello/{id}", printer)
			r.Post("/", printer)
		})
		r.Mount("/test2", chi.NewRouter().Group(func(r chi.Router) {
			r.Get("/inner", printer)
		}))
	})

	return r
}

func TestMain(m *testing.M) {
	testServer = httptest.NewServer(server(8080))
	m.Run()
}

func TestMiddleware_OPTIONSRequest(t *testing.T) {
	tests := []struct {
		path string
		o    describer.Routes
	}{
		{
			path: "/route/test2/inner",
			o:    describer.Routes{describer.RouteInfo{Method: "GET", Path: "/"}},
		},
		{
			path: "/route/test2",
			o:    describer.Routes{describer.RouteInfo{Method: "GET", Path: "/inner"}},
		},
		{
			path: "/route/test",
			o: describer.Routes{
				describer.RouteInfo{Method: "GET", Path: "/hello/{id}"},
				describer.RouteInfo{Method: "POST", Path: "/"},
			},
		},
		{
			path: "/route/test/hello",
			o:    describer.Routes{describer.RouteInfo{Method: "GET", Path: "/{id}"}},
		},
		{
			path: "/route",
			o: describer.Routes{
				describer.RouteInfo{Method: "GET", Path: "/get"},
				describer.RouteInfo{Method: "POST", Path: "/post"}, describer.RouteInfo{Method: "PUT", Path: "/put"},
				describer.RouteInfo{Method: "PATCH", Path: "/patch"}, describer.RouteInfo{Method: "DELETE", Path: "/delete"},
				describer.RouteInfo{Method: "GET", Path: "/test/hello/{id}"}, describer.RouteInfo{Method: "POST", Path: "/test/"},
				describer.RouteInfo{Method: "GET", Path: "/test2/inner"},
			},
		},
		{
			path: "/",
			o: describer.Routes{
				describer.RouteInfo{Method: "GET", Path: "/get"},
				describer.RouteInfo{Method: "POST", Path: "/post"}, describer.RouteInfo{Method: "PUT", Path: "/put"},
				describer.RouteInfo{Method: "PATCH", Path: "/patch"}, describer.RouteInfo{Method: "DELETE", Path: "/delete"},
				describer.RouteInfo{Method: "GET", Path: "/route/get"},
				describer.RouteInfo{Method: "POST", Path: "/route/post"}, describer.RouteInfo{Method: "PUT", Path: "/route/put"},
				describer.RouteInfo{Method: "PATCH", Path: "/route/patch"}, describer.RouteInfo{Method: "DELETE", Path: "/route/delete"},
				describer.RouteInfo{Method: "GET", Path: "/route/test/hello/{id}"}, describer.RouteInfo{Method: "POST", Path: "/route/test/"},
				describer.RouteInfo{Method: "GET", Path: "/route/test2/inner"},
			},
		},
	}

	for i, test := range tests {
		u, err := url.Parse(fmt.Sprintf("%s%s", testServer.URL, test.path))
		assert.NoError(t, err, "test %d: error while parsing url", i)
		v, err := http.DefaultClient.Do(&http.Request{Method: "OPTIONS", URL: u})
		assert.NoError(t, err, "test %d: error while making request", i)

		defer v.Body.Close()
		body, err := ioutil.ReadAll(v.Body)
		assert.NoError(t, err, "test %d: error while reading response body", i)

		var result describer.Routes
		assert.NoError(t, json.Unmarshal(body, &result), "test %d: error while unmarshaling results", i)
		// Sorting both cases to ensure coherence
		sort.Sort(test.o)
		sort.Sort(result)
		assert.Equal(t, test.o, result, "test %d: results mismatch", i)
	}
}

func TestMiddleware_NotOPTIONRequest(t *testing.T) {
	tests := []struct {
		path   string
		method string
	}{
		{path: "/route/test2/inner", method: "GET"},
		{path: "/get", method: "GET"},
		{path: "/post", method: "POST"},
	}

	for i, test := range tests {
		u, err := url.Parse(fmt.Sprintf("%s%s", testServer.URL, test.path))
		assert.NoError(t, err, "test %d: error while parsing url", i)
		v, err := http.DefaultClient.Do(&http.Request{Method: test.method, URL: u})
		assert.NoError(t, err, "test %d: error while making request", i)

		defer v.Body.Close()
		body, err := ioutil.ReadAll(v.Body)
		assert.NoError(t, err, "test %d: error while reading response body", i)
		assert.Equal(t, []byte(test.path), body, "test %d: body mismatch", i)
	}
}
