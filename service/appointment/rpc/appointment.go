package main

import (
	"flag"
	"fmt"

	"hospital/common/authn"
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/config"
	"hospital/service/appointment/rpc/internal/server"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/appointment.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	svcCtx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(err)
	}
	defer svcCtx.Close()
	for _, method := range reportRPCMethodsWithSensitiveContent() {
		zrpc.DontLogContentForMethod(method)
	}

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		appointmentv1.RegisterAppointmentServiceServer(grpcServer, server.NewAppointmentServiceServer(svcCtx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(authn.UnaryServerInterceptor(
		svcCtx.TokenManager,
		svcCtx.AuthorizationVersionValidator,
	))
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}

// 报告 RPC 不进入 go-zero 的请求/响应正文日志，避免医疗内容泄漏。
func reportRPCMethodsWithSensitiveContent() []string {
	return []string{
		appointmentv1.AppointmentService_GetExaminationItemReportTemplate_FullMethodName,
		appointmentv1.AppointmentService_SaveExaminationItemReportTemplate_FullMethodName,
		appointmentv1.AppointmentService_SaveExaminationReportDraft_FullMethodName,
		appointmentv1.AppointmentService_CompleteAndPublishExaminationReport_FullMethodName,
		appointmentv1.AppointmentService_CorrectExaminationReport_FullMethodName,
		appointmentv1.AppointmentService_GetExaminationReport_FullMethodName,
		appointmentv1.AppointmentService_ListExaminationReports_FullMethodName,
		appointmentv1.AppointmentService_ListExaminationReportVersions_FullMethodName,
		appointmentv1.AppointmentService_GetMyExaminationReport_FullMethodName,
		appointmentv1.AppointmentService_ListMyExaminationReports_FullMethodName,
	}
}
