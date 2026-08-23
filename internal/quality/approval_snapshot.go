package quality

import "context"

type ApprovalSnapshot struct {
	OrderID  string
	Approved bool
}

func (s *Service) ApprovalSnapshot(ctx context.Context, orderID string) (ApprovalSnapshot, error) {
	approved, err := s.ApprovedForOrder(ctx, s.Store.DB, orderID)
	if err != nil {
		return ApprovalSnapshot{}, err
	}
	return ApprovalSnapshot{OrderID: orderID, Approved: approved}, nil
}
