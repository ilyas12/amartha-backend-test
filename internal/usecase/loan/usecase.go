package loan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"amartha-backend-test/internal/domain/loan"
	"amartha-backend-test/pkg/id"

	"gorm.io/gorm"
)

type Usecase struct{ repo loan.Repository }

func NewUsecase(r loan.Repository) *Usecase { return &Usecase{repo: r} }

type CreateLoanInput struct {
	BorrowerID string  `json:"borrower_id"`
	Principal  float64 `json:"principal"`
	Rate       float64 `json:"rate"`
	ROI        float64 `json:"roi"`
}

type LoanDTO struct {
	LoanID     string    `json:"loan_id"`
	BorrowerID string    `json:"borrower_id"`
	Principal  float64   `json:"principal"`
	Rate       float64   `json:"rate"`
	ROI        float64   `json:"roi"`
	State      string    `json:"state"`
	CreatedAt  time.Time `json:"created_at"`
}

func (u *Usecase) Create(ctx context.Context, in CreateLoanInput) (*LoanDTO, error) {
	if in.BorrowerID == "" || len(in.BorrowerID) != 32 || in.Principal <= 0 {
		return nil, errors.New("invalid input")
	}

	// Block if the borrower already has a pending (proposed) loan.
	pending, err := u.repo.GetPendingLoanByBorrowerID(ctx, in.BorrowerID)
	switch {
	case err == nil:
		return nil, fmt.Errorf("borrower %s already has a pending loan: %s", in.BorrowerID, pending.LoanID)
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, err
	}

	l := &loan.Loan{
		LoanID:         id.NewID32(),
		BorrowerID:     in.BorrowerID,
		Principal:      in.Principal,
		Rate:           in.Rate,
		ROI:            in.ROI,
		State:          loan.StateProposed,
		StateUpdatedAt: time.Now().UTC(),
	}

	if err := u.repo.Create(ctx, l); err != nil {
		return nil, err
	}

	return &LoanDTO{
		LoanID:     l.LoanID,
		BorrowerID: l.BorrowerID,
		Principal:  l.Principal,
		Rate:       l.Rate,
		ROI:        l.ROI,
		State:      string(l.State),
		CreatedAt:  l.CreatedAt,
	}, nil
}

func (u *Usecase) Get(ctx context.Context, loanID string) (*LoanDTO, error) {
	l, err := u.repo.GetByLoanID(ctx, loanID)
	if err != nil {
		return nil, err
	}
	return &LoanDTO{LoanID: l.LoanID, BorrowerID: l.BorrowerID, Principal: l.Principal, Rate: l.Rate, ROI: l.ROI, State: string(l.State), CreatedAt: l.CreatedAt}, nil
}
