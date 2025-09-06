package approval

import (
	"errors"
	"log"
	"time"

	domainApproval "amartha-backend-test/internal/domain/approval"
	domainLoan "amartha-backend-test/internal/domain/loan"
	"amartha-backend-test/internal/domain/uow"
	"amartha-backend-test/pkg/id"
	"context"

	"gorm.io/gorm"
)

type Usecase struct {
	loanRepo     domainLoan.Repository
	approvalRepo domainApproval.Repository
	uow          uow.UnitOfWork
}

// NewUsecase: pass both repos and a UoW for tx flows.
func NewUsecase(loans domainLoan.Repository, approvals domainApproval.Repository, tx uow.UnitOfWork) *Usecase {
	return &Usecase{loanRepo: loans, approvalRepo: approvals, uow: tx}
}

func (u *Usecase) Approve(ctx context.Context, in ApproveInput) (*ApprovalDTO, error) {
	if u.uow == nil {
		return nil, domainLoan.ErrInvalidTransition
	}
	var dto *ApprovalDTO

	err := u.uow.WithinTx(ctx, func(r uow.Repos) error {
		// Lock loan row for update
		l, err := r.Loans.GetByLoanIDForUpdate(ctx, in.LoanID)
		if err != nil {
			return domainLoan.ErrNotFound
		}

		// State guard: only proposed → approved
		if l.State != domainLoan.StateProposed {
			// If already approved, surface a clear error
			if l.State == domainLoan.StateApproved {
				return domainLoan.ErrAlreadyApproved
			}
			return domainLoan.ErrInvalidTransition
		}

		if _, err := r.Approvals.GetByLoanID(ctx, l.ID); err != nil {
			log.Println(err.Error())
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				// real query error → surface upward
				return err
			}
		} else {
			// found one → already approved (business rule)
			return domainLoan.ErrAlreadyApproved
		}

		// Insert approval
		a := &domainApproval.Approval{
			ApprovalID:          id.NewID32(),
			LoanID:              l.ID, // numeric FK
			PhotoURL:            in.PhotoURL,
			ValidatorEmployeeID: in.ValidatorEmployeeID,
			ApprovalDate:        in.ApprovalDate.UTC(),
		}
		if err := r.Approvals.Create(ctx, a); err != nil {
			return err
		}

		// Update loan → approved
		l.State = domainLoan.StateApproved
		l.StateUpdatedAt = time.Now().UTC()
		if err := r.Loans.Save(ctx, l); err != nil {
			return err
		}

		dto = &ApprovalDTO{
			ApprovalID: a.ApprovalID,
			LoanID:     l.LoanID, // public id
			PhotoURL:   a.PhotoURL,
			ApprovedAt: a.ApprovalDate,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return dto, nil
}
