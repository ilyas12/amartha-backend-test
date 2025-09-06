package mysql

import (
	"amartha-backend-test/internal/domain/loan"
	"amartha-backend-test/internal/domain/uow"
	"context"

	"gorm.io/gorm"
)

type GormUoW struct{ db *gorm.DB }

func NewGormUoW(db *gorm.DB) *GormUoW { return &GormUoW{db: db} }

func (u *GormUoW) WithinTx(ctx context.Context, fn func(r uow.Repos) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		r := uow.Repos{
			Loans:     &LoanRepository{db: tx},
			Approvals: &ApprovalRepository{db: tx},
		}
		return fn(r)
	})
}

func (u *GormUoW) WithinLoanTx(ctx context.Context, loanID string, fn func(r uow.Repos, l *loan.Loan) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		r := uow.Repos{
			Loans:     &LoanRepository{db: tx},
			Approvals: &ApprovalRepository{db: tx},
		}
		// lock the loan row up-front to prevent races
		l, err := r.Loans.GetByLoanIDForUpdate(ctx, loanID)
		if err != nil {
			return err
		}
		return fn(r, l)
	})
}
