package approval

import "context"

type Repository interface {
	// Create a new approval (DB uniqueness ensures at most one per loan)
	Create(ctx context.Context, a *Approval) error

	// Get  approval by loan ID
	GetByLoanID(ctx context.Context, loanID uint64) (*Approval, error)

	// Get by public approval_id
	GetByApprovalID(ctx context.Context, approvalID string) (*Approval, error)
}
