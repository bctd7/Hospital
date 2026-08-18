// Package projectconfiguration 负责把 Appointment 项目事实和 Guidance 规则
// 作为一次完整配置提交。它拥有 TCC 协调、版本校验和补偿流程，但不直接调用
// gRPC 或 MySQL；这些能力分别由 AppointmentParticipant 和 Store 窄接口提供。
package projectconfiguration
