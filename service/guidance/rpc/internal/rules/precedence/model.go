// Package precedence 定义检查项目直接先后关系的领域模型和窄端口。
// 数据库只保存工作人员直接配置的边；间接关系由业务层按需推导。
package precedence

import "time"

type ProjectReference struct {
	ItemID       string
	DepartmentID string
	Name         string
	Status       string
	Version      int64
}

type Rule struct {
	RuleID                  string
	OwnerItemID             string
	PredecessorItemID       string
	PredecessorDepartmentID string
	PredecessorItemName     string
	SuccessorItemID         string
	SuccessorDepartmentID   string
	SuccessorItemName       string
	StaffReason             string
	PatientMessage          string
	CreatedBy               string
	CreateOperationID       string
	Version                 int64
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// Relation 是查询结果。Direct=false 表示它由多条直接规则临时推导，RuleID 为空。
type Relation struct {
	Rule
	Direct     bool
	PathLength int
}

type Direction string

const (
	DirectionAll          Direction = "all"
	DirectionPredecessors Direction = "predecessors"
	DirectionSuccessors   Direction = "successors"
)

func (d Direction) Valid() bool {
	return d == DirectionAll || d == DirectionPredecessors || d == DirectionSuccessors
}
