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
	"hospital/service/app/api/internal/httperror"
	"hospital/service/app/api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/app-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	logx.AddGlobalFields(logx.Field(projectlog.FieldEnvironment, c.Environment))
	httpx.SetErrorHandlerCtx(httperror.Handler)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()
	server.Use(httpaccess.Middleware())

	ctx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(err)
	}
	defer ctx.Close()
	handler.RegisterHandlers(server, ctx)

	projectlog.Info(
		context.Background(),
		projectlog.EventServiceStarting,
		logx.Field(projectlog.FieldHost, c.Host),
		logx.Field(projectlog.FieldPort, c.Port),
	)
	server.Start()
}
