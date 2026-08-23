package reporting

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type CustodyRow struct {
	SampleID   string
	ChainRef   string
	State      domain.QualityState
	Receiver   string
	LastAction string
	LastActor  string
	LastAt     time.Time
}

type CustodyOverview struct {
	Received        int
	Approved        int
	Rejected        int
	MissingReceiver int
	Latest          *time.Time
}

func BuildCustody(rows []CustodyRow) CustodyOverview {
	overview := CustodyOverview{}
	for _, row := range rows {
		switch row.State {
		case domain.QualityReceived:
			overview.Received++
		case domain.QualityApproved:
			overview.Approved++
		case domain.QualityRejected:
			overview.Rejected++
		}
		if strings.TrimSpace(row.Receiver) == "" {
			overview.MissingReceiver++
		}
		if !row.LastAt.IsZero() && (overview.Latest == nil || row.LastAt.After(*overview.Latest)) {
			value := row.LastAt
			overview.Latest = &value
		}
	}
	return overview
}

func SortCustody(rows []CustodyRow) []CustodyRow {
	copyOf := append([]CustodyRow(nil), rows...)
	sort.SliceStable(copyOf, func(i, j int) bool {
		if copyOf[i].LastAt.Equal(copyOf[j].LastAt) {
			return copyOf[i].SampleID < copyOf[j].SampleID
		}
		return copyOf[i].LastAt.Before(copyOf[j].LastAt)
	})
	return copyOf
}

func ValidateCustody(row CustodyRow) error {
	if strings.TrimSpace(row.SampleID) == "" || strings.TrimSpace(row.ChainRef) == "" {
		return domain.ErrInvalid
	}
	if row.State != domain.QualityReceived && row.State != domain.QualityApproved && row.State != domain.QualityRejected {
		return fmt.Errorf("%w: quality state", domain.ErrInvalid)
	}
	if row.State == domain.QualityApproved && (row.LastAction == "" || row.LastActor == "" || row.LastAt.IsZero()) {
		return fmt.Errorf("%w: approved sample lacks custody evidence", domain.ErrConflict)
	}
	return nil
}

func RejectedCustody(rows []CustodyRow) []CustodyRow {
	result := make([]CustodyRow, 0)
	for _, row := range rows {
		if row.State == domain.QualityRejected {
			result = append(result, row)
		}
	}
	return SortCustody(result)
}

func CustodyLabel(row CustodyRow) string {
	if row.State == domain.QualityApproved {
		return fmt.Sprintf("%s approved by %s", row.ChainRef, row.LastActor)
	}
	if row.State == domain.QualityRejected {
		return fmt.Sprintf("%s rejected", row.ChainRef)
	}
	return fmt.Sprintf("%s awaiting review", row.ChainRef)
}

func CustodyComplete(rows []CustodyRow) bool {
	if len(rows) == 0 {
		return false
	}
	for _, row := range rows {
		if ValidateCustody(row) != nil || row.State != domain.QualityApproved {
			return false
		}
	}
	return true
}
