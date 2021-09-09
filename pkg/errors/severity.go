package errors

// severity is type identifier for severity level of error
// lower the value higher the criticality
type severity int

// ISeverity severity interface
type ISeverity interface {
	// Is returns true if the both the severity have the same level
	Is(sev ISeverity) bool

	// String stringer implementation of severity
	String() string

	// Level given the integer value of level
	// lower the value higher the criticality
	Level() int
}

const (
	// SeverityCritical indicate the severity level critical
	// this should be used for high impact errors
	// Ex:
	// * Extreme infrastructure or product outage involving multiple customers
	// * Customer or critical data loss or corruption
	// * Any product is completely unusable or unavailable for 2 or more customers
	// * No workaround available
	SeverityCritical severity = iota

	// SeverityHigh indicate the severity level major
	// this should be used for significant impact errors
	// Ex:
	// * Major functionality/feature degradation of a product for 2 or more customers
	// * Any product is completely unavailable for 1 or more customers
	// * No workaround available
	SeverityHigh

	// SeverityMedium indicate the severity level medium
	// Ex:
	// * Non-critical functional degradation of the product
	// * Does not meet sev 1 or sev 2
	// * Possible short-term workaround available
	SeverityMedium

	// SeverityLow indicate the severity level low
	// this should be used for low or no impact errors
	// Ex:
	// * No current or know customer impact
	// * Engineering escalation might be required
	SeverityLow
)

// severityNames holds the string name for severity
// here names will be in order of severity (higher to lower)
// note: maintaining order is mandatory
var severityNames = [...]string{"Critical", "High", "Medium", "Low"}

// Is returns true if the both the severity have the same level
func (s severity) Is(sev ISeverity) bool {
	return s.Level() == sev.Level()
}

// String stringer implementation of severity
func (s severity) String() string {
	return severityNames[s]
}

// Level given the integer value of level
// lower the value higher the criticality
func (s severity) Level() int {
	return int(s)
}
