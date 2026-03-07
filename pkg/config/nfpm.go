package config

import "github.com/goreleaser/nfpm/v2"

func (alt NFPMIPKAlternative) ToNFP() nfpm.IPKAlternative {
	return nfpm.IPKAlternative{
		Priority: alt.Priority,
		Target:   alt.Target,
		LinkName: alt.LinkName,
	}
}

func (ipk NFPMIPK) ToNFPAlts() []nfpm.IPKAlternative {
	alts := make([]nfpm.IPKAlternative, len(ipk.Alternatives))
	for i, alt := range ipk.Alternatives {
		alts[i] = alt.ToNFP()
	}
	return alts
}
