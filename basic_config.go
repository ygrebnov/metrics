package metrics

type basicProviderConfig struct {
	// when false, remove per-key mutex entries from `inits` after initialization to
	// allow GC of mutexes for many ephemeral instrument names. Default: false.
	doNotCleanupInits bool
	logger            logger
}

// BasicProviderOption configures a BasicProvider constructed by NewBasicProvider.
type BasicProviderOption func(*basicProviderConfig)

// WithInitCleanupDisabled controls whether per-key init mutex entries are removed from
// the provider's internal `inits` map after initialization. When enabled the
// entries are deleted to allow GC of mutexes for ephemeral instrument names.
// Init cleanup is enabled by default; this option disables it.
func WithInitCleanupDisabled() BasicProviderOption {
	return func(cfg *basicProviderConfig) { cfg.doNotCleanupInits = true }
}

func WithBasicProviderLogger(l logger) BasicProviderOption {
	return func(cfg *basicProviderConfig) { cfg.logger = l }
}
