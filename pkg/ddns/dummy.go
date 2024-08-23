package ddns

type ProviderDummy struct{}

func (provider *ProviderDummy) UpdateDomain(domainConfig *DomainConfig) error {
	return nil
}
