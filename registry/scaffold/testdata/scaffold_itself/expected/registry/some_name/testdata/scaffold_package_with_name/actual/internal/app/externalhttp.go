package something

import "github.com/kjuulh/scaffoldhttp"

func externalHttpServer() func() (*http.ExternalServer, error) {
	return func() (*http.ExternalServer, error) {
		return http.NewExternal(), nil
	}
}
