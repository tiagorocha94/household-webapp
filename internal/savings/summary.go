package savings

// Summarize derives the figures of an account from its movements and
// snapshots, given in any order.
func Summarize(a Account, movements []Movement, snapshots []Snapshot) AccountSummary {
	s := AccountSummary{Account: a, MovementCount: len(movements), SnapshotCount: len(snapshots)}
	for _, m := range movements {
		switch m.Type {
		case TypeDeposit:
			s.DepositedCents += m.AmountCents
		case TypeWithdrawal:
			s.WithdrawnCents += m.AmountCents
		}
	}
	net := s.DepositedCents - s.WithdrawnCents

	// The latest snapshot is the one of the latest month, then the highest ID.
	var latest *Snapshot
	for i := range snapshots {
		sn := &snapshots[i]
		if latest == nil || sn.Month > latest.Month || (sn.Month == latest.Month && sn.ID > latest.ID) {
			latest = sn
		}
	}
	s.BalanceCents = net
	if latest != nil {
		s.BalanceCents = latest.ValueCents
		s.InterestCents = latest.ValueCents - net
		s.LastSnapshotMonth = latest.Month
	}

	if a.TargetCents > 0 && s.BalanceCents > 0 {
		s.ProgressPct = int(s.BalanceCents * 100 / a.TargetCents)
	}
	return s
}
