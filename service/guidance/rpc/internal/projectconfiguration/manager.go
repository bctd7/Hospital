package projectconfiguration

import (
	"errors"

	descriptionrules "hospital/service/guidance/rpc/internal/rules/description"
)

// Manager 是完整项目配置的业务入口。
//
// 它只持有三个窄依赖：Guidance 自有存储、Appointment 的 TCC 参与者、
// Appointment 项目只读目录。具体流程按职责拆在 read.go、configure.go、
// transaction.go 和 validation.go 中。
type Manager struct {
	store       Store
	appointment AppointmentParticipant
	projects    ProjectDirectory
	description *descriptionrules.Parser
}

func NewManager(store Store, appointment AppointmentParticipant, projects ProjectDirectory, description *descriptionrules.Parser) (*Manager, error) {
	if store == nil || appointment == nil || projects == nil || description == nil {
		return nil, errors.New("project configuration dependencies are required")
	}
	return &Manager{store: store, appointment: appointment, projects: projects, description: description}, nil
}
