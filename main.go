package main

import (
	"fmt"
	"gateway/internal/config"
	"gateway/internal/controller"
	"gateway/internal/proxy"
	ratelimit "gateway/internal/rate-limit"
	"gateway/internal/request"
	"gateway/internal/services"
	"gateway/internal/validate"
	"os"
	"path/filepath"
)

func main() {
	args := os.Args
	fmt.Print(args[1])
	if len(args) > 0 && args[1] == "read" {
		readInternal("./internal")
		return
	}

	var configLoader config.Loader = config.NewMockConfigLoader()
	var validator validate.Validator = validate.ReqValidator{}
	var proxier proxy.Proxier = proxy.ReqProxy{}
	var rateLimiter ratelimit.RateLimiter = ratelimit.ReqRateLimiter{}
	var serviceAccessController controller.ServiceAccessController = controller.ReqServiceAccessController{}

	request.GetHandlerFunc(request.HandleRequestData{
		ConfigLoader:            configLoader,
		Validator:               validator,
		RateLimiter:             rateLimiter,
		ServiceAccessController: serviceAccessController,
		Proxier:                 proxier,
		GetService: func(scs []config.ServiceConfig) services.Storer {
			return services.NewMockServiceStore(scs)
		},
	})

}

func readInternal(path string) {
	dir, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}

	for _, file := range dir {
		fileName := file.Name()
		fullPath := filepath.Join(path, fileName)
		if !file.IsDir() {
			data, err := os.ReadFile(fullPath)
			if err != nil {
				panic(err)
			}
			fmt.Println("---logging file---", fullPath)
			fmt.Print(string(data))
		} else {
			readInternal(fullPath)
		}

	}
}
