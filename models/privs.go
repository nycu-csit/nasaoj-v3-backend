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
	Privs     Privileges `json:"privs" yaml:"privs"`
	IsEnabled bool       `json:"isEnabled" yaml:"isEnabled"`
}

func NewWorkerPrivs() Privileges {
	return Privileges{
		CanScheduleAPI: ScopeAll,
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
