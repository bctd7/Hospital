// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package main

import (
	"context"
	"flag"

	"hospital/common/observability/httpaccess"
	projectlog "hospital/common/observability/logging"
	"hospital/service/app/api/internal/config"
	"hospital/service/app/api/internal/handler"
	"hospital/service/app/api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/app-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	logx.AddGlobalFields(logx.Field(projectlog.FieldEnvironment, c.Environment))

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()
	server.Use(httpaccess.Middleware())

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	projectlog.Info(
		context.Background(),
		projectlog.EventServiceStarting,
		logx.Field(projectlog.FieldHost, c.Host),
		logx.Field(projectlog.FieldPort, c.Port),
	)
	server.Start()
}
