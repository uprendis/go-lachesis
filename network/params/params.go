package params

// gas settings
const (
	// MaxGasPowerUsed - max value of Gas Power used in one event
	MaxGasPowerUsed = 10000000 + EventGas

	EventGas  = 1000
	ParentGas = 64
	// ExtraDataGas is cost per byte of event payload data
	PayloadDataGas = 1
)
