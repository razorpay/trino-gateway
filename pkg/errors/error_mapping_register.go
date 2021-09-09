package errors

var (
	publicErrorMapping   = make(map[IdentifierCode]IPublic)
	internalErrorMapping = make(map[IdentifierCode]ErrorCode)
)

// IMapper represents the error mapping data which should be loaded into the error package
type IMapper interface {
	GetIdentifierCode() string
	GetInternalErrorCode() string
	GetPublicErrorCode() string
	GetErrorDescription() string
	GetReason() string
	GetFailureType() string
	GetSource() string
	GetStep() string
	GetNextBestAction() string
	GetRecoverable() string
	GetLink() string
}

// IErrorMapLoader interface which has to be implemented by error code mappign library
// to read the mapping data and return the data in required format
type IErrorMapLoader interface {
	ReadFilesIntoStruct(services []string) ([]IMapper, error)
}

// InitMapping will load all the error codes mentioned in the service directory provided
func InitMapping(mapper IErrorMapLoader, services []string) error {
	// read the valid filed and load the data into struct
	errorMapperList, err := mapper.ReadFilesIntoStruct(services)
	if err != nil {
		return err
	}

	for i := 0; i < len(errorMapperList); i++ {
		updateMappingList(errorMapperList[i])
	}

	RegisterMultiple(publicErrorMapping, internalErrorMapping)
	return nil
}

// updateMappingList updates global error mapping for the IMapper provided
// this would also include internal and public error data mapping
func updateMappingList(errorMapper IMapper) {
	public := &Public{
		Code:        errorMapper.GetPublicErrorCode(),
		Description: errorMapper.GetErrorDescription(),
		Source:      errorMapper.GetSource(),
		Step:        errorMapper.GetStep(),
		Reason:      errorMapper.GetReason(),
	}

	publicErrorMapping[IdentifierCode(errorMapper.GetIdentifierCode())] = public

	internalErrorMapping[IdentifierCode(errorMapper.GetIdentifierCode())] = ErrorCode(errorMapper.GetInternalErrorCode())
}
