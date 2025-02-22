package something

import "git.front.kjuulh.io/kjuulh/scaffoldhttp"

func externalHttpServer() func() (*http.ExternalServer, error) {
	return func() (*http.ExternalServer, error) {
		return http.NewExternal(), nil
	}
}
