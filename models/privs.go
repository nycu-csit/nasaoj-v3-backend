package models

// Split it roughly for now, maybe can fine-grained for
// other usage afterwards

var (
	ScopeNo   = 0
	ScopeSelf = 1
	ScopeAll  = 2
)

type Privileges struct {
	OwnerCsId string `json:"ownerCsId"`

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

func NewJobPrivs(owner string) Privileges {
	return Privileges{
		OwnerCsId:        owner,
		CanUserArgsWrite: ScopeSelf,
	}
}

func NewUserPrivs(owner string) Privileges {
	return Privileges{
		OwnerCsId:   owner,
		CanAPIRead:  ScopeSelf,
		CanAPIWrite: ScopeSelf,
	}
}

func NewAdminPrivs(owner string) Privileges {
	return Privileges{
		OwnerCsId:   owner,
		CanAPIRead:  ScopeAll,
		CanAPIWrite: ScopeAll,
	}
}
