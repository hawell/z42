package main

import (
	"flag"
	"fmt"
	"github.com/hawell/z42/internal/handler"
	"github.com/hawell/z42/internal/server"
	"github.com/hawell/z42/internal/storage"
	"github.com/hawell/z42/pkg/ratelimit"
	"github.com/json-iterator/go"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hawell/logger"
	"github.com/miekg/dns"
	_ "net/http/pprof"
)

var (
	servers           []*dns.Server
	redisDataHandler  *storage.DataHandler
	redisStatHandler  *storage.StatHandler
	dnsRequestHandler *handler.DnsRequestHandler
	rateLimiter       *ratelimit.RateLimiter
	configFile        string
)

func main() {
	configPtr := flag.String("c", "config.json", "path to config file")
	verifyPtr := flag.Bool("t", false, "verify configuration")
	generateConfigPtr := flag.String("g", "template-config.json", "generate template config file")

	flag.Parse()
	flagset := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { flagset[f.Name] = true })

	configFile = *configPtr
	if *verifyPtr {
		Verify(configFile)
		return
	}

	if flagset["g"] {
		data, err := jsoniter.MarshalIndent(resolverDefaultConfig, "", "  ")
		if err != nil {
			fmt.Println("cannot unmarshal template config : ", err)
			return
		}
		if err = ioutil.WriteFile(*generateConfigPtr, data, 0644); err != nil {
			fmt.Printf("cannot save template config to file %s : %s\n", *generateConfigPtr, err)
		}
		return
	}

	Start()

	// TODO: this should be part of a general api
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGHUP)

	for sig := range c {
		switch sig {
		case syscall.SIGINT:
			Stop()
			return
		case syscall.SIGHUP:
			Stop()
			Start()
		}
	}
}

func handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	context := handler.NewRequestContext(w, r)
	// logger.Default.Debugf("handle request: [%d] %s %s", r.Id, context.RawName(), context.Type())

	if rateLimiter.CanHandle(context.IP()) {
		dnsRequestHandler.HandleRequest(context)
	} else {
		context.Res = dns.RcodeRefused
		context.Response()
	}
}

func Start() {
	log.Printf("[INFO] loading config : %s", configFile)
	cfg, _ := LoadConfig(configFile)

	log.Printf("[INFO] loading logger...")
	logger.Default = logger.NewLogger(&cfg.ErrorLog, nil)
	log.Printf("[INFO] logger loaded")

	servers = server.NewServer(cfg.Server)

	redisDataHandler = storage.NewDataHandler(&cfg.RedisData)
	redisStatHandler = storage.NewStatHandler(&cfg.RedisStat)

	logger.Default.Info("starting handler...")
	dnsRequestHandler = handler.NewHandler(&cfg.Handler, redisDataHandler)
	logger.Default.Info("handler started")

	rateLimiter = ratelimit.NewRateLimiter(&cfg.RateLimit)

	dns.HandleFunc(".", handleRequest)

	logger.Default.Info("binding listeners...")
	for i := range servers {
		go func(i int) {
			err := servers[i].ListenAndServe()
			if err != nil {
				logger.Default.Errorf("listener error : %s", err)
			}
		}(i)
	}
	logger.Default.Info("binding completed")
}

func Stop() {
	for i := range servers {
		_ = servers[i].Shutdown()
	}
	dnsRequestHandler.ShutDown()
	redisDataHandler.ShutDown()
	redisStatHandler.ShutDown()
}