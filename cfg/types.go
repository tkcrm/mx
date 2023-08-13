package cfg

type configField struct {
	path              string
	envName           string
	defaultValue      string
	usage             string
	example           string
	validateParams    string
	isRequired        bool
	isSecret          bool
	disableValidation bool
}
