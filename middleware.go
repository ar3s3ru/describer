package selfDescribe

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

// DefaultHijackOptions is used as Middleware options if no other
// options are specified.
//
// Default encoding is JSON.
var DefaultHijackOptions = HijackOptions{
	ContentType: "application/json",
	Render:      json.Marshal,
}

// RenderFn is a function that takes the output of the middleware
// and serialize it in a certain encoding.
type RenderFn func(v interface{}) ([]byte, error)

// HijackOptions specifies options for the Middleware middleware, such as:
//
//   - "Content-Type" header and serialization function
//	 - Policy check logic to hide sensible paths from unauthorized users
//
type HijackOptions struct {
	ContentType string
	Render      RenderFn
}

func getStringSliceFromURI(uri string) (v []string) {
	v = strings.Split(uri, "/")
	if uri == "/" {
		// Using slice [1:] because when splitting root ("/")
		// strings.Split will generate 2 empty strings.
		v = v[1:]
	}
	return
}

// Middleware is a middleware that enables automatic routing schema
// self-description, when making an "OPTIONS" HTTP request to the chi.Router
// instance that uses this middleware.
//
// The HTTP Response this middleware produces contains an Routes object.
// See the documentation
//
// Options can be passed through by using an HijackOptions object to this function.
// Keep in mind that only the first passed HijackOptions instance will be used
// as actual options.
func Middleware(options ...HijackOptions) func(http.Handler) http.Handler {
	opt := DefaultHijackOptions
	if len(options) > 0 {
		opt = options[0]
	}
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, logEntry := chi.RouteContext(r.Context()), middleware.GetLogEntry(r)
			if ctx == nil || r.Method != "OPTIONS" {
				// Just proxy to the next handler
				h.ServeHTTP(w, r)
				return
			}
			// Hijack request
			var routes Routes
			u := getStringSliceFromURI(r.RequestURI)
			chi.Walk(ctx.Routes,
				func(m string, r string, h http.Handler, mw ...func(http.Handler) http.Handler) error {
					sr := getStringSliceFromURI(strings.Replace(r, "/*/", "/", -1))
					lr, lu := len(sr), len(u)
					if lr < lu {
						// Current node path is shorter than requested URI path,
						// so it can't possibly be a sub-route.
						return nil
					}
					for i := 0; i < lu; i++ {
						if u[i] != sr[i] {
							// Mismatch means "u" is not contained in "r";
							// we only want to show routes that are at equal or lower
							// level than the request URI.
							return nil
						}
					}
					routes = append(routes, RouteInfo{
						Method: m,
						Path:   fmt.Sprintf("/%s", strings.Join(sr[lu:], "/")),
					})
					return nil
				})
			raw, err := opt.Render(routes)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				logEntry.Panic(fmt.Sprintf("rendering OPTIONS description failed: %s", err), nil)
				return
			}
			w.WriteHeader(200)
			w.Header().Add("Content-Type", opt.ContentType)
			w.Write(raw)
		})
	}
}

// RouteInfo is the representation of a API HTTP Route.
type RouteInfo struct {
	Method      string `json:"method"`
	Path        string `json:"uri"`
	Description string `json:"description,omitempty"`
}

// Routes is the list of all API routes detected from the middlware.
type Routes []RouteInfo

func (r Routes) Len() int {
	return len(r)
}

func (r Routes) Less(i int, j int) bool {
	return strings.Compare(r[i].Path, r[j].Path) == -1
}

func (r Routes) Swap(i int, j int) {
	r[i], r[j] = r[j], r[i]
}
