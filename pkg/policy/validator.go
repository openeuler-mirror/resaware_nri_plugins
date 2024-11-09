package policy

type Validator interface {
	Validate() error
}
