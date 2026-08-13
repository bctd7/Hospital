package input

import staffsupport "hospital/service/appointment/rpc/internal/manager/staff/support"

// Operation 携带一次写操作的幂等编号和链路请求编号。
type Operation struct {
	OperationID string
	RequestID   string
}

// NormalizeOperation 规范化业务输入携带的幂等编号和请求编号。
func NormalizeOperation(value Operation) (Operation, error) {
	operationID, requestID, err := staffsupport.NormalizeOperation(value.OperationID, value.RequestID)
	if err != nil {
		return Operation{}, err
	}
	return Operation{OperationID: operationID, RequestID: requestID}, nil
}
