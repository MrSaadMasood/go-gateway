package route

import (
	"gateway/internal/common"
	"gateway/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouter(t *testing.T) {

	originalEndpoint := "/endpoint"
	redirectionEndpoint := "/redirected"
	endpoint := "/service-one/v1/endpoint"

	var router Router = NewReqRouter()

	contentTypeHeader := "Content-Type"
	routeConfig := config.RouteConfig{
		Headers: map[string]string{
			contentTypeHeader: "text/xml",
		},
		HeaderRegex: map[string]string{
			contentTypeHeader: "xml$",
		},
		Host:      "test.com",
		HostRegex: "ai.com$",
		Methods:   []string{http.MethodGet, http.MethodPost},
		Query: map[string]string{
			"mobile": "true",
		},
		QueryRegex: map[string]string{
			"ip": `\d$`,
		},
		ClientIp:     "1.1.1.1",
		RedirectPath: redirectionEndpoint,
	}

	reqRoutingConfigMap := map[config.ReqPath]config.RouteConfig{
		"/endpoint": routeConfig,
	}

	testTables := []struct {
		name string
		t    func(t *testing.T)
	}{
		{
			name: "shoud route to the original url if nothing matches for redirection",
			t: func(t *testing.T) {

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				route, err := router.Route(r, &config.ServiceReqRoutingOpts{})

				assert.NoError(t, err)
				expectedRoute, _, err := common.RequestServiceExemptedPath(r.URL.Path)
				assert.NoError(t, err)
				assert.Equal(t, expectedRoute, route)

			},
		},
		{
			name: "should route to the provided redirect url if the header matches",
			t: func(t *testing.T) {

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				r.Header.Set(contentTypeHeader, "text/xml")

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap,
				})

				assert.NoError(t, err)
				assert.Equal(t, redirectionEndpoint, route)

			},
		},

		{
			name: "should route to the original url if the header does not match",
			t: func(t *testing.T) {

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				r.Header.Set(contentTypeHeader, "text/html")

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, originalEndpoint, route)

			},
		},
		{
			name: "should route to the provided redirect url if the header regex matches",
			t: func(t *testing.T) {

				endpoint := "/service-one/v1/endpoint"
				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				r.Header.Set(contentTypeHeader, "text/xml")

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, redirectionEndpoint, route)

			},
		},
		{
			name: "should route to original path if the header regex does not match",
			t: func(t *testing.T) {

				endpoint := "/service-one/v1/endpoint"
				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				r.Header.Set(contentTypeHeader, "text/html")

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, originalEndpoint, route)

			},
		},
		{
			name: "should route to the provided redirect url if the host matches",
			t: func(t *testing.T) {

				endpoint := "/service-one/v1/endpoint"
				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				r.Header.Set(contentTypeHeader, "text/xml")
				r.Host = "test.com"

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, redirectionEndpoint, route)

			},
		},
		{
			name: "should route to the original url if the host does not match",
			t: func(t *testing.T) {

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				r.Host = "no.com"

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, originalEndpoint, route)

			},
		},
		{
			name: "should route to the provided redirect url if the host regex matches",
			t: func(t *testing.T) {

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				r.Header.Set(contentTypeHeader, "text/xml")
				r.Host = "random.ai.com"

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, redirectionEndpoint, route)

			},
		},
		{
			name: "should route to the original route if the host regex does not match",
			t: func(t *testing.T) {

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				r.Host = "random.ran.ai"

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, originalEndpoint, route)

			},
		},
		{
			name: "should route to the redirect url if the request method matches",
			t: func(t *testing.T) {

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				r.Header.Set(contentTypeHeader, "text/xml")
				r.Host = "random.ai.com"

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, redirectionEndpoint, route)

			},
		},
		{
			name: "should route to the original url if the request method does not match",
			t: func(t *testing.T) {

				r := httptest.NewRequest(http.MethodPut, endpoint, http.NoBody)

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, originalEndpoint, route)

			},
		},
		{
			name: "should route to the redirect url if the query matches",
			t: func(t *testing.T) {

				r := httptest.NewRequest(http.MethodGet, endpoint+"?mobile=true", http.NoBody)
				r.Header.Set(contentTypeHeader, "text/xml")
				r.Host = "random.ai.com"

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, redirectionEndpoint, route)

			},
		},
		{
			name: "should route to the original url if the query does not match",
			t: func(t *testing.T) {

				r := httptest.NewRequest(http.MethodGet, endpoint+"?mobie=false", http.NoBody)

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, originalEndpoint, route)

			},
		},
		{
			name: "should route to the redirect url if the query regex matches",
			t: func(t *testing.T) {

				r := httptest.NewRequest(http.MethodGet, endpoint+"?ip=1.1.1.1", http.NoBody)
				r.Header.Set(contentTypeHeader, "text/xml")
				r.Host = "random.ai.com"

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, redirectionEndpoint, route)

			},
		},
		{
			name: "should route to the original url if the query regex not match",
			t: func(t *testing.T) {

				r := httptest.NewRequest(http.MethodGet, endpoint+"?ip=a.b.c.d", http.NoBody)

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, originalEndpoint, route)

			},
		},
		{
			name: "should route to the redirect url if the client ip matches",
			t: func(t *testing.T) {

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				r.Header.Set(contentTypeHeader, "text/xml")
				r.Header.Set("X-Forwarded-For", "1.1.1.1")
				r.Host = "random.ai.com"

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, redirectionEndpoint, route)

			},
		},
		{
			name: "should route to the redirect url if the client ip does not match",
			t: func(t *testing.T) {

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				r.Header.Set("X-Forwarded-For", "0.0.0.0")

				route, err := router.Route(r, &config.ServiceReqRoutingOpts{
					ReqRoutingConfigMap: reqRoutingConfigMap})

				assert.NoError(t, err)
				assert.Equal(t, originalEndpoint, route)

			},
		},
	}

	for _, test := range testTables {
		t.Run(test.name, test.t)
	}

}
