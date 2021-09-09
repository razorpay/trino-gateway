package errors

// IClass interface for error class
type IClass interface {
	// Type returns the class eType
	Type() ErrorType

	// Parent will return the base error class
	Parent() IClass

	// severity will return the severity level of error class
	Severity() ISeverity

	// Error makes the class compatible with error interface
	Error() string

	// New returns an Error instance for a specific class.
	New(identifierCode IdentifierCode) IError

	// Is checked if the given class matches any of the class hierarchy
	Is(target IClass) bool
}

// class allows for classification of errors by a eType.
// A default error eCode for the can be set in the Code attribute.
type class struct {
	// severity holds the severity level for this class
	// this will be applicable only to the class applied
	// nested/parent class may have different severity levels
	severity ISeverity

	// class holds the parent error class
	// this is to maintain the hierarchy of the error
	parent IClass

	// eType identifier of the error class
	// error class comparison will happen using this identifier
	eType ErrorType
}

// NewClass creates new instance of class with reference to parent class
// this will maintain the hierarchy of the types for further usage
func NewClass(name ErrorType, severity ISeverity, parent IClass) IClass {
	// when parent class is not defined
	// then use the base class as parent
	if parent == nil {
		parent = class{
			eType: TypeBase,
		}
	}

	return class{
		parent:   parent,
		eType:    name,
		severity: severity,
	}
}

// Type returns the class eType
func (c class) Type() ErrorType {
	return c.eType
}

// Parent will return the base error class
func (c class) Parent() IClass {
	return c.parent
}

// severity will return the severity level of the error class
func (c class) Severity() ISeverity {
	return c.severity
}

// Error makes the class compatible with error interface
func (c class) Error() string {
	return c.eType.String()
}

// New returns an Error instance for a specific class.
func (c class) New(identifierCode IdentifierCode) IError {
	// constructs an error with the given data eCode
	// also get Public data by eCode from map, if registered,
	// and embed into the error
	err := Error{
		class:    c,
		internal: newInternal(identifierCode),
		public:   newPublicFromCode(identifierCode),
	}

	return err
}

// Is checked if the given class matches any of the class hierarchy
// present in the source class
func (c class) Is(target IClass) bool {
	var eClass IClass = c
	for {
		if eClass.Type() == target.Type() {
			return true
		}

		// get the parent class for nested comparison
		if parent := eClass.Parent(); parent != nil {
			eClass = parent
		} else {
			return false
		}
	}
}
