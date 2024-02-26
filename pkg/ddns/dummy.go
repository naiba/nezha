package ddns

type ProviderDummy struct{}

func (provider ProviderDummy) UpdateDomain(domainConfig *DomainConfig) bool {
	return false
}
