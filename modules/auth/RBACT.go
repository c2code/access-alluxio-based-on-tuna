package auth

/*********************Role-Based Access Control of Tenants****************************/

type RbactBaseRequest struct {
	User      string       `json:"user"`     //*
	Domain    string       `json:"domain"`   //*
    Obj       string
    Method    string
}

func (m Manager) rbactInsertPolicy(policy string, user string, domain string, object string, method string) {

	m.rbact.AddPolicy(policy, domain, object, method)

	m.rbact.AddGroupingPolicy(user, policy, domain)

	m.rbact.SavePolicy()
}

func (m Manager) rbactDeletePolicy(policy string, user string, domain string, object string, method string) {

	m.rbact.RemovePolicy(policy, domain, object, method)

	m.rbact.RemoveGroupingPolicy(user, policy, domain)

	m.rbact.SavePolicy()
}

func (m Manager) rbactCheckRights(user string, domain string, object string, method string) bool {

	return m.rbact.Enforce(user, domain, object, method)
}
