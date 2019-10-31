package sql

// Extension to any specific vendor data
type vendorExtension interface {
	// Gets if the extension is compatible with the component name
	IsCompatible(componentName string) bool

	// Complete the missing driver data from the connection string
	ProcessConnectionString(connectionString string, configuration *driverConfiguration)
}

var vendorExtensions []vendorExtension
