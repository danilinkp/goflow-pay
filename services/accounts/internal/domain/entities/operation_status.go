package entities

type OperationType string

const (
	Deposit    OperationType = "deposit"
	Withdrawal OperationType = "withdrawal"
)

func (op OperationType) String() string {
	return string(op)
}

func (op OperationType) IsValid() bool {
	return op == Deposit || op == Withdrawal
}

type OperationStatus string

const (
	PendingStatus OperationStatus = "pending"
	SuccessStatus OperationStatus = "success"
	FailedStatus  OperationStatus = "failed"
)

func (s OperationStatus) IsValid() bool {
	return s == PendingStatus || s == SuccessStatus || s == FailedStatus
}

func (s OperationStatus) String() string {
	return string(s)
}
