package buildinfo

const (
	ProductName        = "Runethread"
	SourceRepository   = "runethread/core"
	ManagedMetadataDir = ".runethread"
	// ReleaseVersion identifies the running product/binary release.
	ReleaseVersion = "v0.7.0"
	// ContractReleaseVersion identifies the immutable official release that owns
	// the embedded memory-repository control plane. Advance it only when that
	// control plane changes; executable-only releases keep the existing anchor.
	ContractReleaseVersion   = "v0.7.0"
	RepositoryFormatVersion  = 2
	SchemaVersion            = 1
	ContractVersion          = 7
	IndexFormatVersion       = 2
	TrustLockVersion         = 2
	BootstrapVerifierVersion = "v0.6.0"
)
