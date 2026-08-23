package domain

import (
	"strings"
	"time"
)

type Actor struct {
	ID       string
	TenantID string
	Email    string
	Role     string
}

type Vessel struct {
	ID          string
	TenantID    string
	IMO         string
	Name        string
	Flag        string
	DeadweightT float64
	Status      string
	Certificate Certificate
}

type Certificate struct {
	Number    string
	ExpiresAt time.Time
	Verified  bool
}

type Terminal struct {
	ID        string
	TenantID  string
	Name      string
	Timezone  string
	OpenFrom  string
	OpenUntil string
	Status    string
}

type FuelLot struct {
	ID          string
	TenantID    string
	LotNumber   string
	Product     string
	AvailableKG float64
	Quality     QualityState
	ReceivedAt  time.Time
}

type BunkerWindow struct {
	ID         string
	TenantID   string
	TerminalID string
	StartsAt   time.Time
	EndsAt     time.Time
	Status     string
	OwnerID    string
	Version    int64
}

type TransferOrder struct {
	ID            string
	TenantID      string
	VesselID      string
	WindowID      string
	FuelLotID     string
	TargetKG      float64
	TransferredKG float64
	State         OperationState
	LeaseOwner    string
	LeaseUntil    time.Time
	Version       int64
}

type TransferStep struct {
	ID          string
	OrderID     string
	Position    int
	Name        string
	Status      string
	ConfirmedAt *time.Time
}

type SafetyPermit struct {
	ID       string
	OrderID  string
	Status   string
	IssuedBy string
}

type Sample struct {
	ID       string
	OrderID  string
	ChainRef string
	Receiver string
	State    QualityState
	History  []CustodyEvent
}

type CustodyEvent struct {
	ID        string
	SampleID  string
	ActorID   string
	Action    string
	CreatedAt time.Time
}

type Invoice struct {
	ID         string
	OrderID    string
	Amount     int64
	Currency   string
	State      string
	PaymentKey string
}

type OutboxMessage struct {
	ID          string
	Topic       string
	Payload     string
	Status      string
	Attempts    int
	NextAttempt time.Time
	LeaseOwner  string
	LeaseUntil  time.Time
}

type AuditEvent struct {
	ID        string
	TenantID  string
	ActorID   string
	Action    string
	ObjectID  string
	RequestID string
	CreatedAt time.Time
}

type Incident struct {
	ID       string
	TenantID string
	OrderID  string
	Severity string
	Status   string
	Summary  string
}

func NormalizeIMO(value string) string { return strings.ToUpper(strings.TrimSpace(value)) }

func NormalizeLotNumber(value string) string { return strings.ToUpper(strings.TrimSpace(value)) }

func ValidateIMO(value string) bool {
	value = NormalizeIMO(value)
	if len(value) != 7 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (c Certificate) Active(now time.Time) bool { return c.Verified && c.ExpiresAt.After(now) }

func (t Terminal) Accepts(now time.Time) bool {
	return t.Status == "active" && !now.IsZero()
}
