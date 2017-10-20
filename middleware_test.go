package describer_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"testing"
	"time"

	"github.com/ar3s3ru/describer"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	go runServer(8080)
	<-time.After(100 * time.Millisecond) // Just to be sure the server is running

	m.Run()
}

func TestMiddleware(t *testing.T) {
	checkURLs := func(t *testing.T, s string) *url.URL {
		u, err := url.Parse(s)
		assert.NoError(t, err)
		return u
	}

	tests := []struct {
		i *http.Request
		o describer.Routes
	}{
		{
			i: &http.Request{Method: "OPTIONS", URL: checkURLs(t, "http://localhost:8080/route/test2/inner")},
			o: describer.Routes{describer.RouteInfo{Method: "GET", Path: "/"}},
		},
		{
			i: &http.Request{Method: "OPTIONS", URL: checkURLs(t, "http://localhost:8080/route/test2")},
			o: describer.Routes{describer.RouteInfo{Method: "GET", Path: "/inner"}},
		},
		{
			i: &http.Request{Method: "OPTIONS", URL: checkURLs(t, "http://localhost:8080/route/test")},
			o: describer.Routes{
				describer.RouteInfo{Method: "GET", Path: "/hello/{id}"},
				describer.RouteInfo{Method: "POST", Path: "/"},
			},
		},
		{
			i: &http.Request{Method: "OPTIONS", URL: checkURLs(t, "http://localhost:8080/route/test/hello")},
			o: describer.Routes{describer.RouteInfo{Method: "GET", Path: "/{id}"}},
		},
		{
			i: &http.Request{Method: "OPTIONS", URL: checkURLs(t, "http://localhost:8080/route")},
			o: describer.Routes{
				describer.RouteInfo{Method: "GET", Path: "/get"},
				describer.RouteInfo{Method: "POST", Path: "/post"}, describer.RouteInfo{Method: "PUT", Path: "/put"},
				describer.RouteInfo{Method: "PATCH", Path: "/patch"}, describer.RouteInfo{Method: "DELETE", Path: "/delete"},
				describer.RouteInfo{Method: "GET", Path: "/test/hello/{id}"}, describer.RouteInfo{Method: "POST", Path: "/test/"},
				describer.RouteInfo{Method: "GET", Path: "/test2/inner"},
			},
		},
		{
			i: &http.Request{Method: "OPTIONS", URL: checkURLs(t, "http://localhost:8080")},
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
		v, err := http.DefaultClient.Do(test.i)
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

// HTTP server used in TestselfDescribe
func runServer(port int) {
	helloWorld := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello world!"))
	}

	r := chi.NewRouter()
	r.Use(describer.Middleware())

	r.Get("/get", helloWorld)
	r.Post("/post", helloWorld)
	r.Put("/put", helloWorld)
	r.Patch("/patch", helloWorld)
	r.Delete("/delete", helloWorld)

	r.Route("/route", func(r chi.Router) {
		r.Get("/get", helloWorld)
		r.Post("/post", helloWorld)
		r.Put("/put", helloWorld)
		r.Patch("/patch", helloWorld)
		r.Delete("/delete", helloWorld)
		r.Route("/test", func(r chi.Router) {
			r.Get("/hello/{id}", helloWorld)
			r.Post("/", helloWorld)
		})
		r.Mount("/test2", chi.NewRouter().Group(func(r chi.Router) {
			r.Get("/inner", helloWorld)
		}))
	})

	log.Print(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}
