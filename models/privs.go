package models

// Split it roughly for now, maybe can fine-grained for
// other usage afterwards

const (
	ScopeNo   = 0
	ScopeSelf = 1
	ScopeAll  = 2
)

type Privileges struct {
	OwnerId int64 `json:"ownerId"`

	CanAPIRead       int `json:"canAPIRead"`
	CanAPIWrite      int `json:"canAPIWrite"`
	CanUserArgsWrite int `json:"canUserArgsWrite"`
	CanScheduleAPI   int `json:"canScheduleAPI"`
}

type Token struct {
	// UserId binds the token to a user: Privs/namespace membership are
	// resolved live from that user on every request instead of the values
	// below, which are then just informational. Zero for static
	// (admin/worker) tokens, where Privs is authoritative.
	UserId int64 `json:"userId,omitempty" yaml:"userId,omitempty"`

	// Description is required at issuance time so `judge admin get tokens`
	// can tell tokens apart without decoding the secret itself.
	Description string `json:"description" yaml:"description"`

	Privs     Privileges `json:"privs" yaml:"privs"`
	IsEnabled bool       `json:"isEnabled" yaml:"isEnabled"`
	CreatedAt int64      `json:"createdAt" yaml:"createdAt"`
}

func NewWorkerPrivs() Privileges {
	return Privileges{
		// Workers forward put-args / put-score requests on behalf of jobs, so
		// they need user-args write in addition to the schedule API. The actual
		// target is verified by the submission-queue ownership check.
		CanScheduleAPI:   ScopeAll,
		CanUserArgsWrite: ScopeAll,
	}
}

func NewJobPrivs(owner int64) Privileges {
	return Privileges{
		OwnerId:          owner,
		CanUserArgsWrite: ScopeSelf,
	}
}

func NewUserPrivs(owner int64) Privileges {
	return Privileges{
		OwnerId:     owner,
		CanAPIRead:  ScopeSelf,
		CanAPIWrite: ScopeSelf,
	}
}

func NewAdminPrivs(owner int64) Privileges {
	return Privileges{
		OwnerId:     owner,
		CanAPIRead:  ScopeAll,
		CanAPIWrite: ScopeAll,
	}
}
